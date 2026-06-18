import { Ref, onMounted, onUnmounted } from 'vue'
import { Terminal } from '@xterm/xterm'
import { WebglAddon } from '@xterm/addon-webgl'
import { FitAddon } from '@xterm/addon-fit'
import { EventsOn, EventsOff, EventsEmit } from '../../wailsjs/runtime'
import { StartPty, ResizePty } from '../../wailsjs/go/main/App'

export function useTerminal(containerRef: Ref<HTMLElement | undefined>) {
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: '"Cascadia Code", "Consolas", "Courier New", monospace',
    theme: {
      background: '#1e1e2e',
      foreground: '#cdd6f4',
      cursor: '#f5e0dc',
      selectionBackground: '#45475a',
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
    },
  })

  const fitAddon = new FitAddon()
  let resizeObserver: ResizeObserver | null = null

  onMounted(async () => {
    if (!containerRef.value) return

    term.loadAddon(fitAddon)
    term.open(containerRef.value)

    try {
      term.loadAddon(new WebglAddon())
    } catch {
      // fallback to canvas renderer
    }

    fitAddon.fit()

    EventsOn('pty:data', (data: string) => {
      term.write(data)
    })

    term.onData(data => EventsEmit('pty:input', data))

    try {
      await StartPty(term.cols, term.rows)
    } catch (e) {
      term.write(`\r\n\x1b[31m启动 PowerShell 失败: ${e}\x1b[0m\r\n`)
    }

    resizeObserver = new ResizeObserver(() => {
      fitAddon.fit()
      ResizePty(term.cols, term.rows)
    })
    resizeObserver.observe(containerRef.value)
  })

  onUnmounted(() => {
    resizeObserver?.disconnect()
    EventsOff('pty:data')
    term.dispose()
  })

  return { term }
}
