//go:build windows

package main

// 离屏 WebView2 → 单页可选文字 PDF。
//
// 为什么自己手写 COM：go-webview2 v1.0.22 的 pkg/webview2 包虽然封装了 PrintToPdf，
// 但它的 package init 在较新 Go 上 panic（compileCallback: argument size is larger
// than uintptr），一旦 import 整个 app 启动即崩。因此这里只 import 安全的 pkg/edge
// （Wails 自身在用），自定义最小 comProc + 镜像所需 vtable 布局（字段顺序照抄
// pkg/webview2 源码），并自己做 QueryInterface，全部 NewCallback 延迟到运行时。

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

// ── 最小 COM 基建（自包含，不依赖 pkg/webview2）────────────────────────────────

type comProc uintptr

func (p comProc) Call(a ...uintptr) (uintptr, uintptr, error) {
	return syscall.SyscallN(uintptr(p), a...)
}

func newComProc(fn interface{}) comProc {
	return comProc(windows.NewCallback(fn))
}

type iUnknownVtbl struct {
	QueryInterface comProc
	AddRef         comProc
	Release        comProc
}

// 以下 vtable 结构字段顺序照抄 go-webview2/pkg/webview2 源码，保证槽位一致。

type printSettingsVtbl struct {
	iUnknownVtbl
	GetOrientation                comProc
	PutOrientation                comProc
	GetScaleFactor                comProc
	PutScaleFactor                comProc
	GetPageWidth                  comProc
	PutPageWidth                  comProc
	GetPageHeight                 comProc
	PutPageHeight                 comProc
	GetMarginTop                  comProc
	PutMarginTop                  comProc
	GetMarginBottom               comProc
	PutMarginBottom               comProc
	GetMarginLeft                 comProc
	PutMarginLeft                 comProc
	GetMarginRight                comProc
	PutMarginRight                comProc
	GetShouldPrintBackgrounds     comProc
	PutShouldPrintBackgrounds     comProc
	GetShouldPrintSelectionOnly   comProc
	PutShouldPrintSelectionOnly   comProc
	GetShouldPrintHeaderAndFooter comProc
	PutShouldPrintHeaderAndFooter comProc
	GetHeaderTitle                comProc
	PutHeaderTitle                comProc
	GetFooterUri                  comProc
	PutFooterUri                  comProc
}
type printSettings struct{ vtbl *printSettingsVtbl }

type webview7Vtbl struct {
	iUnknownVtbl
	PrintToPdf comProc
}
type webview7 struct{ vtbl *webview7Vtbl }

type environment6Vtbl struct {
	iUnknownVtbl
	CreatePrintSettings comProc
}
type environment6 struct{ vtbl *environment6Vtbl }

type pdfHandlerVtbl struct {
	iUnknownVtbl
	Invoke comProc
}
type pdfHandler struct{ vtbl *pdfHandlerVtbl }

// 导出全程串行（pdfExportMu），故用包级变量在回调间传递完成结果。
var (
	pdfExportMu        sync.Mutex
	currentPdfDone     func(errorCode uintptr, ok bool)
	pdfHandlerVtblOnce sync.Once
	pdfHandlerVtblInst pdfHandlerVtbl
)

func pdfHandlerQI(_, _, _ uintptr) uintptr { return 0x80004002 } // E_NOINTERFACE
func pdfHandlerAddRef(_ uintptr) uintptr   { return 1 }
func pdfHandlerRelease(_ uintptr) uintptr  { return 1 }
func pdfHandlerInvoke(_, errorCode, isSuccessful uintptr) uintptr {
	if currentPdfDone != nil {
		currentPdfDone(errorCode, isSuccessful != 0)
	}
	return 0
}

func newPdfHandler() *pdfHandler {
	pdfHandlerVtblOnce.Do(func() {
		pdfHandlerVtblInst = pdfHandlerVtbl{
			iUnknownVtbl{
				newComProc(pdfHandlerQI),
				newComProc(pdfHandlerAddRef),
				newComProc(pdfHandlerRelease),
			},
			newComProc(pdfHandlerInvoke),
		}
	})
	return &pdfHandler{vtbl: &pdfHandlerVtblInst}
}

// comQueryInterface 对一个 COM 接口指针（首字 = vtbl 指针）调用 IUnknown::QueryInterface(槽0)。
func comQueryInterface(this unsafe.Pointer, iid *windows.GUID) (unsafe.Pointer, error) {
	vt := *(*uintptr)(this)               // vtbl 指针
	qi := *(*uintptr)(unsafe.Pointer(vt)) // vtbl[0] = QueryInterface
	var out unsafe.Pointer
	hr, _, _ := syscall.SyscallN(qi,
		uintptr(this),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 || out == nil {
		return nil, fmt.Errorf("QueryInterface 失败: hr=0x%x", hr)
	}
	return out, nil
}

// putDoubleByValue / putBoolByValue：绕过 pkg/webview2 中 Put* 把 double/bool 按指针
// 传的 BUG，直接调 vtable 槽位、按值传（double 用 math.Float64bits）。
func putDoubleByValue(slot comProc, this unsafe.Pointer, v float64) error {
	hr, _, _ := slot.Call(uintptr(this), uintptr(math.Float64bits(v)))
	if hr != 0 {
		return fmt.Errorf("put double 失败: hr=0x%x", hr)
	}
	return nil
}

func putBoolByValue(slot comProc, this unsafe.Pointer, v bool) error {
	var b uintptr
	if v {
		b = 1
	}
	hr, _, _ := slot.Call(uintptr(this), b)
	if hr != 0 {
		return fmt.Errorf("put bool 失败: hr=0x%x", hr)
	}
	return nil
}

// ── Win32 隐藏窗口 + 消息泵 ───────────────────────────────────────────────────

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW = modUser32.NewProc("RegisterClassExW")
	procUnregisterClassW = modUser32.NewProc("UnregisterClassW")
	procCreateWindowExW  = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow    = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW   = modUser32.NewProc("DefWindowProcW")
	procGetMessageW      = modUser32.NewProc("GetMessageW")
	procTranslateMessage = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage  = modUser32.NewProc("PostQuitMessage")

	procGetModuleHandleW = modKernel32.NewProc("GetModuleHandleW")
)

const _WS_POPUP = 0x80000000

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

type msgW struct {
	hwnd    windows.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

var offscreenClassSeq uint64

func getModuleHandle() windows.Handle {
	h, _, _ := procGetModuleHandleW.Call(0)
	return windows.Handle(h)
}

// printHTMLToPDF 用离屏 WebView2 把自包含 html 导出为单页 PDF 到 outPath。
// pageWIn/pageHIn 单位英寸；scale 为缩放系数。整段在专属 LockOSThread 线程上跑，同步返回。
func printHTMLToPDF(html, outPath string, pageWIn, pageHIn, scale float64) error {
	pdfExportMu.Lock()
	defer pdfExportMu.Unlock()

	resultCh := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		resultCh <- runOffscreenPrint(html, outPath, pageWIn, pageHIn, scale)
	}()
	return <-resultCh
}

func runOffscreenPrint(html, outPath string, pageWIn, pageHIn, scale float64) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("离屏 PDF 导出 panic: %v", r)
		}
	}()

	iid7, err := windows.GUIDFromString("{79c24d83-09a3-45ae-9418-487f32a58740}") // ICoreWebView2_7
	if err != nil {
		return err
	}
	iidEnv6, err := windows.GUIDFromString("{e59ee362-acbd-4857-9a8e-d3644d9459a9}") // ICoreWebView2Environment6
	if err != nil {
		return err
	}

	// 1) 注册唯一隐藏窗口类。
	hInstance := getModuleHandle()
	className := "STOffscreenPdf_" + strconv.FormatUint(atomicAddSeq(), 10)
	classNamePtr, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return err
	}
	wndProc := newComProc(func(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	})
	wc := wndClassExW{
		lpfnWndProc:   uintptr(wndProc),
		hInstance:     hInstance,
		lpszClassName: classNamePtr,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	atom, _, e1 := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW 失败: %v", e1)
	}
	defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(classNamePtr)), uintptr(hInstance))

	// 2) 创建隐藏窗口（WS_POPUP，不 ShowWindow）。窗口尺寸不决定 PDF 尺寸（PrintToPdf 按 PageWidth 排版）。
	titlePtr, _ := windows.UTF16PtrFromString("offscreen-pdf")
	hwnd, _, e2 := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		_WS_POPUP,
		0, 0, 794, 1123,
		0, 0,
		uintptr(hInstance),
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW 失败: %v", e2)
	}
	defer procDestroyWindow.Call(hwnd)

	// 3) 离屏 Chromium，独立 DataPath 避免与 Wails 自身 WebView2 用户目录冲突。
	chromium := edge.NewChromium()
	chromium.DataPath = filepath.Join(os.TempDir(), "SimpleTerminal-pdf-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	chromium.SetErrorCallback(func(err error) {
		fmt.Fprintf(os.Stderr, "[offscreen-pdf][webview2] %v\n", err)
	})

	var printErr error
	finish := func(e error) {
		printErr = e
		procPostQuitMessage.Call(0)
	}
	currentPdfDone = func(errorCode uintptr, ok bool) {
		if !ok {
			finish(fmt.Errorf("PrintToPdf 失败, errorCode=0x%x", errorCode))
			return
		}
		finish(nil)
	}

	// keep-alive：PrintToPdf 异步，handler/settings/path 必须活到完成回调触发。
	var (
		keepHandler  *pdfHandler
		keepSettings *printSettings
		keepPath     *uint16
		keepWv7      *webview7
	)

	chromium.NavigationCompletedCallback = func(sender *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
		wv7Ptr, err := comQueryInterface(unsafe.Pointer(sender), &iid7)
		if err != nil {
			finish(fmt.Errorf("QI ICoreWebView2_7（运行时过旧？）: %w", err))
			return
		}
		wv7 := (*webview7)(wv7Ptr)
		keepWv7 = wv7

		edgeEnv := chromium.Environment()
		if edgeEnv == nil {
			finish(fmt.Errorf("WebView2 environment 为空"))
			return
		}
		env6Ptr, err := comQueryInterface(unsafe.Pointer(edgeEnv), &iidEnv6)
		if err != nil {
			finish(fmt.Errorf("QI ICoreWebView2Environment6（运行时过旧？）: %w", err))
			return
		}
		env6 := (*environment6)(env6Ptr)

		var setPtr unsafe.Pointer
		if hr, _, _ := env6.vtbl.CreatePrintSettings.Call(
			uintptr(unsafe.Pointer(env6)),
			uintptr(unsafe.Pointer(&setPtr)),
		); hr != 0 || setPtr == nil {
			finish(fmt.Errorf("CreatePrintSettings 失败: hr=0x%x", hr))
			return
		}
		settings := (*printSettings)(setPtr)
		keepSettings = settings
		sp := unsafe.Pointer(settings)

		// 纵向；页宽高/缩放/边距按值传（绕过库 BUG）；打印背景色；不要页眉页脚。
		if hr, _, _ := settings.vtbl.PutOrientation.Call(uintptr(sp), 0); hr != 0 {
			finish(fmt.Errorf("PutOrientation 失败: hr=0x%x", hr))
			return
		}
		for _, s := range []struct {
			slot comProc
			v    float64
		}{
			{settings.vtbl.PutPageWidth, pageWIn},
			{settings.vtbl.PutPageHeight, pageHIn},
			{settings.vtbl.PutScaleFactor, scale},
			{settings.vtbl.PutMarginTop, 0},
			{settings.vtbl.PutMarginBottom, 0},
			{settings.vtbl.PutMarginLeft, 0},
			{settings.vtbl.PutMarginRight, 0},
		} {
			if err := putDoubleByValue(s.slot, sp, s.v); err != nil {
				finish(err)
				return
			}
		}
		if err := putBoolByValue(settings.vtbl.PutShouldPrintBackgrounds, sp, true); err != nil {
			finish(err)
			return
		}
		if err := putBoolByValue(settings.vtbl.PutShouldPrintHeaderAndFooter, sp, false); err != nil {
			finish(err)
			return
		}

		pathPtr, err := windows.UTF16PtrFromString(outPath)
		if err != nil {
			finish(err)
			return
		}
		keepPath = pathPtr
		handler := newPdfHandler()
		keepHandler = handler

		if hr, _, _ := wv7.vtbl.PrintToPdf.Call(
			uintptr(unsafe.Pointer(wv7)),
			uintptr(unsafe.Pointer(pathPtr)),
			uintptr(unsafe.Pointer(settings)),
			uintptr(unsafe.Pointer(handler)),
		); hr != 0 {
			finish(fmt.Errorf("PrintToPdf 调用失败: hr=0x%x", hr))
			return
		}
		// 成功发起，等待完成回调（在消息泵里触发）。
	}

	// 4) Embed 同步泵消息直到 webview 就绪。
	chromium.Embed(hwnd)
	// 5) 加载自包含 HTML。
	chromium.NavigateToString(html)

	// 6) 消息泵：导航完成→设置→PrintToPdf；完成回调 PostQuitMessage 退出。
	var msg msgW
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 { // WM_QUIT(0) 或错误(-1)
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	runtime.KeepAlive(keepHandler)
	runtime.KeepAlive(keepSettings)
	runtime.KeepAlive(keepPath)
	runtime.KeepAlive(keepWv7)
	return printErr
}

func atomicAddSeq() uint64 {
	// 串行导出下无需原子，但保留唯一性 + 时间戳避免类名复用冲突。
	offscreenClassSeq++
	return offscreenClassSeq*1_000_000_000 + uint64(time.Now().UnixNano()%1_000_000_000)
}
