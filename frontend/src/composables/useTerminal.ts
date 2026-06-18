import { Terminal } from '@xterm/xterm'
import type { ITheme } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { CanvasAddon } from '@xterm/addon-canvas'
import '@xterm/xterm/css/xterm.css'
import { EventsOn, EventsEmit } from '../../wailsjs/runtime'
import { StartPty, ResizePty } from '../../wailsjs/go/main/App'

// Catppuccin Mocha 终端配色
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

export function useTerminal() {
  let term: Terminal | null = null
  let fitAddon: FitAddon | null = null
  let resizeObserver: ResizeObserver | null = null
  let cleanupData: (() => void) | null = null

  function mount(el: HTMLElement) {
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

    // WebGL 渲染（不花屏）；上下文丢失或不支持时回退 Canvas
    loadRenderer(term)

    fitAddon.fit()

    // Go → 前端：原始终端输出
    cleanupData = EventsOn('pty:data', (data: string) => {
      term?.write(data)
    })
    // 前端 → Go：键盘输入/粘贴
    term.onData((data) => EventsEmit('pty:input', data))

    // 启动 PTY 会话，初始尺寸用 fit 后的 cols/rows
    void StartPty(term.cols, term.rows)

    // 容器尺寸变化（分隔线拖动/窗口缩放）→ fit → 同步 PTY 尺寸
    resizeObserver = new ResizeObserver(() => {
      if (!term || !fitAddon) return
      fitAddon.fit()
      void ResizePty(term.cols, term.rows)
    })
    resizeObserver.observe(el)
  }

  function loadRenderer(t: Terminal) {
    try {
      const webgl = new WebglAddon()
      webgl.onContextLoss(() => {
        webgl.dispose()
        try {
          t.loadAddon(new CanvasAddon())
        } catch {
          /* 回退失败则使用默认 DOM 渲染 */
        }
      })
      t.loadAddon(webgl)
    } catch {
      try {
        t.loadAddon(new CanvasAddon())
      } catch {
        /* 使用默认 DOM 渲染 */
      }
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

  return { mount, getTerm, dispose }
}
