//go:build windows

package main

// 离屏 WebView2 → 单页可选文字 PDF / 长图 PNG（走 Chrome DevTools 协议）。
//
// 为什么用 CDP 而不是 ICoreWebView2_7.PrintToPdf：
//  1. go-webview2 的 pkg/webview2 包 init 在新版 Go 上 panic，不能 import。
//  2. 该包里 ICoreWebView2_7 / Environment6 的 vtable 结构是错的（把继承链接口当成只有
//     IUnknown+1 个方法，真实 vtable 是累积的），照抄会调错槽位直接崩。
// 因此只用 pkg/edge（Wails 自身在用、手写正确）的基类 ICoreWebView2，其 vtable 里
// CallDevToolsProtocolMethod 槽位正确。本文件照抄 edge 的基类 vtable 布局到该方法为止，
// 按字段名访问（Go 自动算偏移）。
//
// PDF（四步）：setEmulatedMedia(print) → setDeviceMetricsOverride(794) →
//   getLayoutMetrics(精确高度) → printToPDF（单页紧凑、printBackground、矢量文字可选）。
//   打印媒体排版让量高与打印一致，避免最后一行溢出第二页。
// PNG 长图（三步，屏幕媒体）：setDeviceMetricsOverride(794) → getLayoutMetrics →
//   captureScreenshot(clip 全文、captureBeyondViewport)。

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

// 离屏导出全程串行（exportMu），故用包级变量在回调间传递完成态。
var (
	exportMu       sync.Mutex
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

// 排版/视口固定宽度 = paperWidth(794px = A4@96dpi)。
const cdpMetricsParams = `{"width":794,"height":1123,"deviceScaleFactor":1,"mobile":false,"screenWidth":794,"screenHeight":1123}`

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

// 离屏隐藏窗口的 WndProc：用单例，避免每次导出都 NewCallback(永不释放→撞回调上限)。
var (
	offscreenWndProcOnce sync.Once
	offscreenWndProc     comProc
)

func getOffscreenWndProc() comProc {
	offscreenWndProcOnce.Do(func() {
		offscreenWndProc = newComProc(func(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
			ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
			return ret
		})
	})
	return offscreenWndProc
}

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

// cdpCall 发起一次 CallDevToolsProtocolMethod，完成时把结果 JSON 交给 cb。
type cdpCall func(method, params string, cb func(resultJSON string, errorCode uintptr))

// runOffscreenLocked 在专属 STA 线程上跑 fn（WebView2 要求 STA 套间 + 自己 CoInitialize）。
func runOffscreenLocked(fn func() error) error {
	exportMu.Lock()
	defer exportMu.Unlock()

	resultCh := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
			pdfLog("CoInitializeEx ret=%v (S_FALSE/已初始化可忽略)", err)
		}
		defer windows.CoUninitialize()
		resultCh <- fn()
	}()
	return <-resultCh
}

// withOffscreenWebView 建隐藏窗口 + 离屏 Chromium，加载 html，导航完成后回调
// build(wv, call, finish) 让调用方发起 CDP 链；finish(err) 结束并退出消息泵。
func withOffscreenWebView(html, tag string, build func(wv *cdpWebView2, call cdpCall, finish func(error))) (retErr error) {
	_ = os.WriteFile(pdfLogPath, []byte("=== "+tag+" export start ===\n"), 0644)
	pdfLog("htmlLen=%d", len(html))
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("离屏导出 panic: %v", r)
		}
		pdfLog("RETURN err=%v", retErr)
	}()

	// 清理上次导出残留的临时 DataPath，避免累积。
	if olds, _ := filepath.Glob(filepath.Join(os.TempDir(), "SimpleTerminal-pdf-*")); olds != nil {
		for _, d := range olds {
			_ = os.RemoveAll(d)
		}
	}

	hInstance := getModuleHandle()
	className := "STOffscreenPdf_" + strconv.FormatUint(atomicAddSeq(), 10)
	classNamePtr, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return err
	}
	wc := wndClassExW{
		lpfnWndProc:   uintptr(getOffscreenWndProc()),
		hInstance:     hInstance,
		lpszClassName: classNamePtr,
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	atom, _, e1 := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW 失败: %v", e1)
	}
	defer procUnregisterClassW.Call(uintptr(unsafe.Pointer(classNamePtr)), uintptr(hInstance))

	titlePtr, _ := windows.UTF16PtrFromString("offscreen-export")
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

	chromium := edge.NewChromium()
	chromium.DataPath = filepath.Join(os.TempDir(), "SimpleTerminal-pdf-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	chromium.SetErrorCallback(func(err error) {
		fmt.Fprintf(os.Stderr, "[offscreen-export][webview2] %v\n", err)
	})

	var resultErr error
	finish := func(e error) {
		resultErr = e
		procPostQuitMessage.Call(0)
	}

	// keep-alive：CDP 调用异步，handler/参数串必须活到对应完成回调触发。
	var (
		keepHandler *cdpHandler
		keepMethod  *uint16
		keepParams  *uint16
	)
	callCDP := func(wv *cdpWebView2, method, params string, cb func(string, uintptr)) {
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

	chromium.NavigationCompletedCallback = func(sender *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
		pdfLog("NavigationCompleted")
		wv := (*cdpWebView2)(unsafe.Pointer(sender))
		build(wv, func(method, params string, cb func(string, uintptr)) {
			callCDP(wv, method, params, cb)
		}, finish)
	}

	pdfLog("before Embed")
	chromium.Embed(hwnd)
	pdfLog("after Embed; NavigateToString")
	chromium.NavigateToString(html)
	pdfLog("entering message loop")

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
	return resultErr
}

// getLayoutHeightPx 解析 Page.getLayoutMetrics 结果里的内容高度(px)。
func getLayoutHeightPx(resultJSON string) float64 {
	var m struct {
		CSSContentSize struct {
			Height float64 `json:"height"`
		} `json:"cssContentSize"`
		ContentSize struct {
			Height float64 `json:"height"`
		} `json:"contentSize"`
	}
	_ = json.Unmarshal([]byte(resultJSON), &m)
	h := m.CSSContentSize.Height
	if h == 0 {
		h = m.ContentSize.Height
	}
	return h
}

// decodeWriteCDPData 从 CDP 结果 JSON 取 base64 data 解码写盘。
func decodeWriteCDPData(resultJSON, outPath string) error {
	var r struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(resultJSON), &r); err != nil {
		return fmt.Errorf("解析结果失败: %w (resp=%.160s)", err, resultJSON)
	}
	if r.Data == "" {
		return fmt.Errorf("结果无 data (resp=%.200s)", resultJSON)
	}
	b, err := base64.StdEncoding.DecodeString(r.Data)
	if err != nil {
		return fmt.Errorf("base64 解码失败: %w", err)
	}
	if err := os.WriteFile(outPath, b, 0644); err != nil {
		return fmt.Errorf("写文件失败: %w", err)
	}
	pdfLog("written %d bytes -> %s", len(b), outPath)
	return nil
}

// printHTMLToPDF 把自包含 html 导出为单页可选文字 PDF。
// pageWIn 页宽(英寸)；maxHIn 单页最大高度(英寸，避开 PDF 200in 硬上限)。
func printHTMLToPDF(html, outPath string, pageWIn, maxHIn float64) error {
	return runOffscreenLocked(func() error {
		return withOffscreenWebView(html, "pdf", func(wv *cdpWebView2, call cdpCall, finish func(error)) {
			// 1) 打印媒体排版（让量高与打印一致）
			call("Emulation.setEmulatedMedia", `{"media":"print"}`, func(_ string, ec uintptr) {
				if ec != 0 {
					finish(fmt.Errorf("setEmulatedMedia errorCode=0x%x", ec))
					return
				}
				// 2) 固定排版宽度
				call("Emulation.setDeviceMetricsOverride", cdpMetricsParams, func(_ string, ec2 uintptr) {
					if ec2 != 0 {
						finish(fmt.Errorf("setDeviceMetricsOverride errorCode=0x%x", ec2))
						return
					}
					// 3) 量精确高度
					call("Page.getLayoutMetrics", "{}", func(js string, ec3 uintptr) {
						if ec3 != 0 {
							finish(fmt.Errorf("getLayoutMetrics errorCode=0x%x", ec3))
							return
						}
						heightPx := getLayoutHeightPx(js)
						if heightPx <= 0 {
							finish(fmt.Errorf("getLayoutMetrics 高度为 0 (resp=%.160s)", js))
							return
						}
						contentIn := (math.Ceil(heightPx) + 8) / 96 // +8px 吸收取整
						scale := 1.0
						paperH := contentIn
						if contentIn > maxHIn {
							scale = maxHIn / contentIn
							paperH = maxHIn
						}
						pdfLog("pdf heightPx=%.1f -> paperH=%.3fin scale=%.4f", heightPx, paperH, scale)
						// 4) 打印
						params := fmt.Sprintf(
							`{"landscape":false,"printBackground":true,"paperWidth":%.4f,"paperHeight":%.4f,`+
								`"marginTop":0,"marginBottom":0,"marginLeft":0,"marginRight":0,"scale":%.4f,"preferCSSPageSize":false}`,
							pageWIn, paperH, scale,
						)
						call("Page.printToPDF", params, func(rj string, ec4 uintptr) {
							if ec4 != 0 {
								finish(fmt.Errorf("printToPDF errorCode=0x%x", ec4))
								return
							}
							finish(decodeWriteCDPData(rj, outPath))
						})
					})
				})
			})
		})
	})
}

func atomicAddSeq() uint64 {
	offscreenClassSeq++
	return offscreenClassSeq*1_000_000_000 + uint64(time.Now().UnixNano()%1_000_000_000)
}
