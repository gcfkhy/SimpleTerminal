import { Terminal } from '@xterm/xterm'
import type { ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { CanvasAddon } from '@xterm/addon-canvas'
import '@xterm/xterm/css/xterm.css'
import { EventsOn, EventsEmit } from '../../wailsjs/runtime'
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

    // 监听当前 Tab 的 PTY 输出：写入终端，同时解析提示符以追踪 cwd 变化
    cleanupData = EventsOn(`pty:data:${id}`, (data: string) => {
      term?.write(data)
      detectCwd(data, onCwdChange)
    })

    // 键盘输入发送 (tabId, data) 两个参数
    term.onData((data) => EventsEmit('pty:input', id, data))

    // 启动 PTY，完成后发送初始 cd
    void StartPty(id, term.cols, term.rows).then(() => {
      if (initialDir) {
        EventsEmit('pty:input', id, `cd "${initialDir}"\r`)
      }
    })

    resizeObserver = new ResizeObserver(() => {
      if (!term || !fitAddon || el.offsetWidth === 0) return
      fitAddon.fit()
      void ResizePty(id, term.cols, term.rows)
    })
    resizeObserver.observe(el)
  }

  // 切换到此 Tab 时由外部调用，确保终端尺寸正确
  function fit() {
    if (!fitAddon || !term) return
    fitAddon.fit()
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
