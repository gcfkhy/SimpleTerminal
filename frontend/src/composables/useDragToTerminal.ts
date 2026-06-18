import type { Terminal } from '@xterm/xterm'

// 把文件树拖来的路径粘贴到终端光标处；含空格的路径自动加双引号。
export function useDragToTerminal(el: HTMLElement, getTerm: () => Terminal | null): () => void {
  function onDragOver(e: DragEvent) {
    // 必须 preventDefault，否则不会触发 drop
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
  }

  function onDrop(e: DragEvent) {
    e.preventDefault()
    const filePath = e.dataTransfer?.getData('text/path')
    if (!filePath) return
    const term = getTerm()
    if (!term) return
    const safePath = filePath.includes(' ') ? `"${filePath}"` : filePath
    term.paste(safePath)
    term.focus()
  }

  el.addEventListener('dragover', onDragOver)
  el.addEventListener('drop', onDrop)

  return () => {
    el.removeEventListener('dragover', onDragOver)
    el.removeEventListener('drop', onDrop)
  }
}
