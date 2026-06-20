//go:build windows

package main

// 离屏 WebView2 → 单页可选文字 PDF（走 Chrome DevTools 协议）。
//
// 为什么用 CDP 而不是 ICoreWebView2_7.PrintToPdf：
//  1. go-webview2 的 pkg/webview2 包 init 在新版 Go 上 panic，不能 import。
//  2. 该包里 ICoreWebView2_7 / Environment6 的 vtable 结构是错的（把继承链接口当成只有
//     IUnknown+1 个方法，真实 vtable 是累积的），照抄会调错槽位直接崩。
// 因此只用 pkg/edge（Wails 自身在用、手写正确）的基类 ICoreWebView2，其 vtable 里
// CallDevToolsProtocolMethod 槽位正确。本文件照抄 edge 的基类 vtable 布局到该方法为止，
// 按字段名访问（Go 自动算偏移）。
//
// 四步 CDP（导航完成后）：
//  1. Emulation.setEmulatedMedia media=print —— 让离屏页以「打印媒体」排版，
//     使后续量到的高度与 printToPDF 的实际排版完全一致（否则 screen 媒体略矮、最后一行溢出第二页）。
//  2. Emulation.setDeviceMetricsOverride width=794 —— 固定排版宽度 = paperWidth。
//  3. Page.getLayoutMetrics —— 量精确内容高度。
//  4. Page.printToPDF —— 按该高度出单页：紧凑、printBackground 保留主题底色、矢量文字可选，
//     base64 返回后写盘。

import (
	"encoding/base64"
	"encoding/json"
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

// cdpWebView2Vtbl：照抄 go-webview2/pkg/edge 的基类 iCoreWebView2Vtbl 字段顺序，
// 仅到 CallDevToolsProtocolMethod 为止。按字段名访问保证槽位正确。
type cdpWebView2Vtbl struct {
	iUnknownVtbl
	GetSettings                            comProc
	GetSource                              comProc
	Navigate                               comProc
	NavigateToString                       comProc
	AddNavigationStarting                  comProc
	RemoveNavigationStarting               comProc
	AddContentLoading                      comProc
	RemoveContentLoading                   comProc
	AddSourceChanged                       comProc
	RemoveSourceChanged                    comProc
	AddHistoryChanged                      comProc
	RemoveHistoryChanged                   comProc
	AddNavigationCompleted                 comProc
	RemoveNavigationCompleted              comProc
	AddFrameNavigationStarting             comProc
	RemoveFrameNavigationStarting          comProc
	AddFrameNavigationCompleted            comProc
	RemoveFrameNavigationCompleted         comProc
	AddScriptDialogOpening                 comProc
	RemoveScriptDialogOpening              comProc
	AddPermissionRequested                 comProc
	RemovePermissionRequested              comProc
	AddProcessFailed                       comProc
	RemoveProcessFailed                    comProc
	AddScriptToExecuteOnDocumentCreated    comProc
	RemoveScriptToExecuteOnDocumentCreated comProc
	ExecuteScript                          comProc
	CapturePreview                         comProc
	Reload                                 comProc
	PostWebMessageAsJSON                   comProc
	PostWebMessageAsString                 comProc
	AddWebMessageReceived                  comProc
	RemoveWebMessageReceived               comProc
	CallDevToolsProtocolMethod             comProc
}
type cdpWebView2 struct{ vtbl *cdpWebView2Vtbl }

// CDP 方法完成回调：ICoreWebView2CallDevToolsProtocolMethodCompletedHandler。
type cdpHandlerVtbl struct {
	iUnknownVtbl
	Invoke comProc
}
type cdpHandler struct{ vtbl *cdpHandlerVtbl }

// 导出全程串行（pdfExportMu），故用包级变量在回调间传递。
var (
	pdfExportMu    sync.Mutex
	cdpDone        func(errorCode uintptr, jsonPtr *uint16)
	cdpHandlerOnce sync.Once
	cdpHandlerInst cdpHandlerVtbl
)

func cdpHandlerQI(_, _, _ uintptr) uintptr { return 0x80004002 } // E_NOINTERFACE
func cdpHandlerAddRef(_ uintptr) uintptr   { return 1 }
func cdpHandlerRelease(_ uintptr) uintptr  { return 1 }
func cdpHandlerInvoke(_, errorCode uintptr, jsonPtr *uint16) uintptr {
	if cdpDone != nil {
		cdpDone(errorCode, jsonPtr)
	}
	return 0
}

func newCdpHandler() *cdpHandler {
	cdpHandlerOnce.Do(func() {
		cdpHandlerInst = cdpHandlerVtbl{
			iUnknownVtbl{
				newComProc(cdpHandlerQI),
				newComProc(cdpHandlerAddRef),
				newComProc(cdpHandlerRelease),
			},
			newComProc(cdpHandlerInvoke),
		}
	})
	return &cdpHandler{vtbl: &cdpHandlerInst}
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

// pdfLog 把离屏导出的每一步追加写到临时日志，便于人工排查。
var pdfLogPath = filepath.Join(os.TempDir(), "st-pdf-debug.log")

func pdfLog(format string, a ...interface{}) {
	f, err := os.OpenFile(pdfLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", a...)
}

// printHTMLToPDF 用离屏 WebView2 把自包含 html 导出为单页 PDF 到 outPath。
// pageWIn 为页宽(英寸)；maxHIn 为单页最大高度(英寸，避开 PDF 200in 硬上限)，
// 内容高度由离屏页(打印媒体)精确测得，超过 maxHIn 时自动按比例缩放。
func printHTMLToPDF(html, outPath string, pageWIn, maxHIn float64) error {
	pdfExportMu.Lock()
	defer pdfExportMu.Unlock()

	resultCh := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		// WebView2 要求创建/调用它的线程是 STA 套间。离屏线程是新建的，必须自己 CoInitializeEx。
		if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
			pdfLog("CoInitializeEx ret=%v (S_FALSE/已初始化可忽略)", err)
		}
		defer windows.CoUninitialize()
		resultCh <- runOffscreenPrint(html, outPath, pageWIn, maxHIn)
	}()
	return <-resultCh
}

func runOffscreenPrint(html, outPath string, pageWIn, maxHIn float64) (retErr error) {
	_ = os.WriteFile(pdfLogPath, []byte("=== pdf export start ===\n"), 0644)
	pdfLog("params w=%.3f maxH=%.3f out=%s htmlLen=%d", pageWIn, maxHIn, outPath, len(html))
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("离屏 PDF 导出 panic: %v", r)
		}
		pdfLog("RETURN err=%v", retErr)
	}()

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

	// 2) 创建隐藏窗口（WS_POPUP，不 ShowWindow）。
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
	pdfLog("window created hwnd=0x%x", hwnd)

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

	// keep-alive：CDP 调用异步，handler/参数串必须活到对应完成回调触发。
	var (
		keepHandler *cdpHandler
		keepMethod  *uint16
		keepParams  *uint16
	)

	// callCDP 发起一次 CallDevToolsProtocolMethod，完成时把结果 JSON 交给 cb。串行调用，故全局态安全。
	callCDP := func(wv *cdpWebView2, method, params string, cb func(resultJSON string, errorCode uintptr)) {
		cdpDone = func(errorCode uintptr, jsonPtr *uint16) {
			s := ""
			if jsonPtr != nil {
				s = windows.UTF16PtrToString(jsonPtr)
			}
			cb(s, errorCode)
		}
		methodPtr, _ := windows.UTF16PtrFromString(method)
		paramsPtr, _ := windows.UTF16PtrFromString(params)
		handler := newCdpHandler()
		keepHandler, keepMethod, keepParams = handler, methodPtr, paramsPtr
		hr, _, _ := wv.vtbl.CallDevToolsProtocolMethod.Call(
			uintptr(unsafe.Pointer(wv)),
			uintptr(unsafe.Pointer(methodPtr)),
			uintptr(unsafe.Pointer(paramsPtr)),
			uintptr(unsafe.Pointer(handler)),
		)
		if hr != 0 {
			finish(fmt.Errorf("CallDevToolsProtocolMethod(%s) 失败: hr=0x%x", method, hr))
		}
	}

	// 第四步：按精确高度打印。
	doPrint := func(wv *cdpWebView2, paperH, scale float64) {
		params := fmt.Sprintf(
			`{"landscape":false,"printBackground":true,"paperWidth":%.4f,"paperHeight":%.4f,`+
				`"marginTop":0,"marginBottom":0,"marginLeft":0,"marginRight":0,"scale":%.4f,"preferCSSPageSize":false}`,
			pageWIn, paperH, scale,
		)
		callCDP(wv, "Page.printToPDF", params, func(resJSON string, ec uintptr) {
			if ec != 0 {
				finish(fmt.Errorf("printToPDF errorCode=0x%x", ec))
				return
			}
			var r struct {
				Data string `json:"data"`
			}
			if err := json.Unmarshal([]byte(resJSON), &r); err != nil {
				finish(fmt.Errorf("解析 printToPDF 结果失败: %w (resp=%.160s)", err, resJSON))
				return
			}
			if r.Data == "" {
				finish(fmt.Errorf("printToPDF 结果无 data (resp=%.200s)", resJSON))
				return
			}
			pdfBytes, err := base64.StdEncoding.DecodeString(r.Data)
			if err != nil {
				finish(fmt.Errorf("base64 解码失败: %w", err))
				return
			}
			if err := os.WriteFile(outPath, pdfBytes, 0644); err != nil {
				finish(fmt.Errorf("写 PDF 文件失败: %w", err))
				return
			}
			pdfLog("PDF written, %d bytes", len(pdfBytes))
			finish(nil)
		})
	}

	// 第三步：量精确内容高度（此时已是打印媒体 + 固定宽度），算出 paperHeight/scale 后打印。
	doMeasure := func(wv *cdpWebView2) {
		callCDP(wv, "Page.getLayoutMetrics", "{}", func(resultJSON string, ec uintptr) {
			if ec != 0 {
				finish(fmt.Errorf("getLayoutMetrics errorCode=0x%x", ec))
				return
			}
			var m struct {
				CSSContentSize struct {
					Height float64 `json:"height"`
				} `json:"cssContentSize"`
				ContentSize struct {
					Height float64 `json:"height"`
				} `json:"contentSize"`
			}
			_ = json.Unmarshal([]byte(resultJSON), &m)
			heightPx := m.CSSContentSize.Height
			if heightPx == 0 {
				heightPx = m.ContentSize.Height
			}
			if heightPx <= 0 {
				finish(fmt.Errorf("getLayoutMetrics 高度为 0 (resp=%.160s)", resultJSON))
				return
			}
			// 打印媒体下量得高度已与打印一致，仅留极小缓冲(8px)吸收取整。
			contentIn := (math.Ceil(heightPx) + 8) / 96
			scale := 1.0
			paperH := contentIn
			if contentIn > maxHIn {
				scale = maxHIn / contentIn
				paperH = maxHIn
			}
			pdfLog("layout heightPx=%.1f -> paperH=%.3fin scale=%.4f", heightPx, paperH, scale)
			doPrint(wv, paperH, scale)
		})
	}

	chromium.NavigationCompletedCallback = func(sender *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
		pdfLog("NavigationCompleted; CDP setEmulatedMedia=print")
		wv := (*cdpWebView2)(unsafe.Pointer(sender))

		// 第一步：切到打印媒体排版。
		callCDP(wv, "Emulation.setEmulatedMedia", `{"media":"print"}`, func(_ string, ec1 uintptr) {
			if ec1 != 0 {
				finish(fmt.Errorf("setEmulatedMedia errorCode=0x%x", ec1))
				return
			}
			pdfLog("media=print ok; setDeviceMetricsOverride")
			// 第二步：固定排版宽度 = paperWidth (794px)。
			callCDP(wv, "Emulation.setDeviceMetricsOverride",
				`{"width":794,"height":1123,"deviceScaleFactor":1,"mobile":false,"screenWidth":794,"screenHeight":1123}`,
				func(_ string, ec2 uintptr) {
					if ec2 != 0 {
						finish(fmt.Errorf("setDeviceMetricsOverride errorCode=0x%x", ec2))
						return
					}
					pdfLog("metrics override ok; getLayoutMetrics")
					doMeasure(wv)
				})
		})
	}

	// 4) Embed 同步泵消息直到 webview 就绪。
	pdfLog("before Embed")
	chromium.Embed(hwnd)
	pdfLog("after Embed; NavigateToString")
	// 5) 加载自包含 HTML。
	chromium.NavigateToString(html)
	pdfLog("NavigateToString called; entering message loop")

	// 6) 消息泵：导航→设打印媒体→固定宽→量高→打印→写盘→PostQuitMessage 退出。
	var msg msgW
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 { // WM_QUIT(0) 或错误(-1)
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	pdfLog("message loop exited")

	runtime.KeepAlive(keepHandler)
	runtime.KeepAlive(keepMethod)
	runtime.KeepAlive(keepParams)
	return printErr
}

func atomicAddSeq() uint64 {
	offscreenClassSeq++
	return offscreenClassSeq*1_000_000_000 + uint64(time.Now().UnixNano()%1_000_000_000)
}
