import { Terminal } from '@xterm/xterm'
import type { ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { CanvasAddon } from '@xterm/addon-canvas'
import '@xterm/xterm/css/xterm.css'
import { EventsOn, EventsEmit, ClipboardSetText } from '../../wailsjs/runtime'
import { StartPty, ResizePty } from '../../wailsjs/go/main/App'

const catppuccinMocha: ITheme = {
  background: '#1e1e2e',
  foreground: '#cdd6f4',
  cursor: '#f5e0dc',
  cursorAccent: '#1e1e2e',
  selectionBackground: '#585b70',
  black: '#45475a',
  red: '#f38ba8',
  green: '#a6e3a1',
  yellow: '#f9e2af',
  blue: '#89b4fa',
  magenta: '#f5c2e7',
  cyan: '#94e2d5',
  white: '#bac2de',
  brightBlack: '#585b70',
  brightRed: '#f38ba8',
  brightGreen: '#a6e3a1',
  brightYellow: '#f9e2af',
  brightBlue: '#89b4fa',
  brightMagenta: '#f5c2e7',
  brightCyan: '#94e2d5',
  brightWhite: '#a6adc8',
}

export function useTerminal(id: string) {
  let term: Terminal | null = null
  let fitAddon: FitAddon | null = null
  let resizeObserver: ResizeObserver | null = null
  let cleanupData: (() => void) | null = null
  // 缩放节流状态：rAF 合并每帧多次回调，防抖 ConPTY resize，记录上次发出的 cols/rows
  let resizeRaf = 0
  let lastCols = 0
  let lastRows = 0
  let ptyResizeTimer: ReturnType<typeof setTimeout> | undefined

  // 从 PowerShell 默认提示符 `PS <路径>>` 解析终端当前工作目录。
  // 直接观察 shell 输出的提示符（经 pty:data 回流）比捕获键盘输入可靠得多——
  // 后者会被粘贴(括号粘贴整段以 ESC 起始)、↑历史/Tab 补全(文本由 PSReadLine 在
  // shell 侧注入、不经 onData)绕过。提示符则反映真实 cwd，对所有输入方式都成立。
  let cwdTail = ''
  let lastCwd: string | null = null
  function detectCwd(data: string, onCwdChange?: (path: string) => void) {
    if (!onCwdChange) return
    // 滚动尾缓冲：提示符可能被分块切断，留一段上下文保证能完整匹配
    cwdTail = (cwdTail + data).slice(-2048)
    // 去除 ANSI/OSC 转义（窗口标题、光标/颜色控制），只留纯文本再匹配
    const plain = cwdTail
      .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '') // OSC（如设置窗口标题）
      .replace(/\x1b[@-Z\\-_]/g, '')                     // 双字符转义
      .replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, '')         // CSI（颜色/光标移动）
    // 取最后一个提示符里的盘符路径（PS E:\... 、PS C:\... 等）
    const re = /PS\s+([A-Za-z]:\\[^\r\n>]*?)\s*>/g
    let m: RegExpExecArray | null
    let found: string | null = null
    while ((m = re.exec(plain)) !== null) found = m[1].replace(/\s+$/, '')
    if (found === null) return
    if (lastCwd === null) { lastCwd = found; return } // 首个提示符=启动目录，静默记录为基准
    if (found !== lastCwd) {
      lastCwd = found
      onCwdChange(found)
    }
  }

  function mount(el: HTMLElement, initialDir?: string, onCwdChange?: (path: string) => void) {
    term = new Terminal({
      fontFamily: '"SF Mono", Consolas, "MiSans", monospace',
      fontSize: 14,
      cursorBlink: true,
      theme: catppuccinMocha,
      allowProposedApi: true,
    })

    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(el)
    loadRenderer(term)
    fitAddon.fit()
    lastCols = term.cols
    lastRows = term.rows

    // 监听当前 Tab 的 PTY 输出：写入终端，同时解析提示符以追踪 cwd 变化
    cleanupData = EventsOn(`pty:data:${id}`, (data: string) => {
      term?.write(data)
      detectCwd(data, onCwdChange)
    })

    // Ctrl+V / Ctrl+C 快捷键处理（仅 keydown、仅 Ctrl 且非 Alt 时介入）。
    term.attachCustomKeyEventHandler((e) => {
      if (e.type !== 'keydown' || !e.ctrlKey || e.altKey) return true

      // Ctrl+V 粘贴：xterm 默认把 Ctrl+V 当控制字符 \x16(SYN)发给程序。PowerShell 的
      // PSReadLine 自己把 \x16 绑成了读剪贴板粘贴所以能用，但 claude 等 TUI 不认 \x16，
      // 表现为“按 Ctrl+V 没反应”。返回 false 让 xterm 既不发 \x16 也不 preventDefault，
      // 浏览器随即触发原生 paste 事件，由 xterm 按 bracketed paste 正确处理——与右键菜单
      // 粘贴同一条路径（已验证在 claude 中可用）。
      if (e.key === 'v' || e.key === 'V') return false

      // Ctrl+C 复制：终端里 Ctrl+C 默认是中断信号(\x03)，不能直接抢走。仅当有选中文本
      // 时复制选区（用 Wails Go 侧剪贴板，打包后的 wails:// 非安全上下文也可靠），并返回
      // false 不发 \x03；没有选中时返回 true，照常发中断信号以打断程序。与 Windows
      // Terminal 行为一致。
      if (e.key === 'c' || e.key === 'C') {
        const sel = term?.getSelection()
        if (sel) {
          void ClipboardSetText(sel)
          return false
        }
        return true
      }

      return true
    })

    // 键盘输入发送 (tabId, data) 两个参数
    term.onData((data) => EventsEmit('pty:input', id, data))

    // 启动 PTY，完成后发送初始 cd
    void StartPty(id, term.cols, term.rows).then(() => {
      if (initialDir) {
        EventsEmit('pty:input', id, `cd "${initialDir}"\r`)
      }
    })

    // 缩放/拖拽时 ResizeObserver 会高频触发。用 rAF 把同一帧内的多次回调合并成一次
    // fit()；ConPTY resize 较重，仅在 cols/rows 真正变化时、并做尾部防抖后再发 IPC，
    // 避免每帧洪泛渲染与 Wails→Go 调用导致卡顿。
    resizeObserver = new ResizeObserver(() => {
      if (resizeRaf) return
      resizeRaf = requestAnimationFrame(() => {
        resizeRaf = 0
        if (!term || !fitAddon || el.offsetWidth === 0) return
        fitAddon.fit()
        if (term.cols === lastCols && term.rows === lastRows) return
        lastCols = term.cols
        lastRows = term.rows
        clearTimeout(ptyResizeTimer)
        ptyResizeTimer = setTimeout(() => {
          if (term) void ResizePty(id, term.cols, term.rows)
        }, 80)
      })
    })
    resizeObserver.observe(el)
  }

  // 切换到此 Tab 时由外部调用，确保终端尺寸正确
  // 切换到此 Tab 时由外部调用：离散事件，直接 fit；仅在尺寸真变化时同步 PTY。
  function fit() {
    if (!fitAddon || !term) return
    fitAddon.fit()
    if (term.cols === lastCols && term.rows === lastRows) return
    lastCols = term.cols
    lastRows = term.rows
    void ResizePty(id, term.cols, term.rows)
  }

  function loadRenderer(t: Terminal) {
    try {
      const webgl = new WebglAddon()
      webgl.onContextLoss(() => {
        webgl.dispose()
        try { t.loadAddon(new CanvasAddon()) } catch { /* DOM fallback */ }
      })
      t.loadAddon(webgl)
    } catch {
      try { t.loadAddon(new CanvasAddon()) } catch { /* DOM fallback */ }
    }
  }

  function getTerm(): Terminal | null {
    return term
  }

  function dispose() {
    if (resizeRaf) cancelAnimationFrame(resizeRaf)
    resizeRaf = 0
    clearTimeout(ptyResizeTimer)
    resizeObserver?.disconnect()
    resizeObserver = null
    cleanupData?.()
    cleanupData = null
    term?.dispose()
    term = null
    fitAddon = null
  }

  return { mount, getTerm, dispose, fit }
}
