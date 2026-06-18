import { Ref, onMounted } from 'vue'
import { Terminal } from '@xterm/xterm'

export function useDragToTerminal(
  containerRef: Ref<HTMLElement | undefined>,
  term: Terminal
) {
  onMounted(() => {
    const el = containerRef.value
    if (!el) return

    el.addEventListener('dragover', (e) => {
      e.preventDefault()
      e.stopPropagation()
    })

    el.addEventListener('drop', (e) => {
      e.preventDefault()
      e.stopPropagation()
      const path = e.dataTransfer?.getData('text/path')
      if (!path) return
      const safe = path.includes(' ') ? `"${path}"` : path
      term.paste(safe)
      term.focus()
    })
  })
}
