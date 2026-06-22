// 从 vscode-office resource/markdown/outline.js 移植纯逻辑，适配 TypeScript + SimpleTerminal。

export interface OutlineItem {
  level: number
  text: string
  id: string
  el: HTMLElement
}

export interface OutlineNode {
  level: number
  text: string
  id: string
  el: HTMLElement | null
  children: OutlineNode[]
}

/**
 * 扫描 .md-body 内的 h1..h6 → 扁平项；过滤掉无文本或无 id 的标题
 * （无 id 的标题无法跳转/高亮）。
 */
export function extractHeadings(bodyEl: HTMLElement | null): OutlineItem[] {
  if (!bodyEl) return []
  const hs = Array.from(bodyEl.querySelectorAll<HTMLElement>('h1,h2,h3,h4,h5,h6'))
  return hs
    .map((el) => ({
      level: parseInt(el.tagName.charAt(1), 10),
      text: (el.textContent || '').trim(),
      id: el.id,
      el,
    }))
    .filter((it) => it.text && it.id)
}

/**
 * 扁平标题列表 → 嵌套树。用栈维护祖先链：弹出所有 level >= 当前的栈顶，
 * 余下栈顶即父；空栈则作顶层。跳级（h1→h3）时 h3 仍挂到最近的更浅标题下。
 */
export function buildOutlineTree(items: OutlineItem[]): OutlineNode[] {
  const roots: OutlineNode[] = []
  const stack: OutlineNode[] = []
  ;(items || []).forEach((raw) => {
    const node: OutlineNode = {
      level: raw.level,
      text: raw.text,
      id: raw.id,
      el: raw.el ?? null,
      children: [],
    }
    while (stack.length && stack[stack.length - 1].level >= node.level) stack.pop()
    if (stack.length) stack[stack.length - 1].children.push(node)
    else roots.push(node)
    stack.push(node)
  })
  return roots
}
