// 从 vscode-office resource/markdown/find.js 移植纯逻辑，适配 TypeScript。

export interface MatchRange {
  start: number
  end: number
}
export interface TextSeg {
  node: Text
  start: number
}
export interface TextIndex {
  full: string
  segs: TextSeg[]
}

// 单次搜索匹配数上限：极大文档防卡死；达到时调用方应提示已截断。
export const MAX_MATCHES = 5000

function clamp(v: number, lo: number, hi: number): number {
  return v < lo ? lo : v > hi ? hi : v
}

/**
 * 在长文本中找出 query 的全部不重叠匹配，返回全局偏移区间 [{start,end}]（end 不含）。
 * 大小写不敏感时优先走快路径：整串 toLowerCase 后长度不变（绝大多数文本），偏移 1:1 对应。
 * 极少数 Unicode 折叠会改变长度（如 'İ'），走慢路径：逐字符折叠 + 反查映射回原串偏移。
 */
export function findMatches(full: string, query: string, caseSensitive: boolean): MatchRange[] {
  const out: MatchRange[] = []
  if (!full || !query) return out

  if (caseSensitive) {
    const step = query.length
    let from = 0
    let at: number
    while ((at = full.indexOf(query, from)) !== -1) {
      out.push({ start: at, end: at + step })
      from = at + step
      if (out.length >= MAX_MATCHES) break
    }
    return out
  }

  const needle = query.toLowerCase()
  const step = needle.length
  if (!step) return out

  const lowFull = full.toLowerCase()
  if (lowFull.length === full.length) {
    // 快路径
    let from = 0
    let at: number
    while ((at = lowFull.indexOf(needle, from)) !== -1) {
      out.push({ start: at, end: at + step })
      from = at + step
      if (out.length >= MAX_MATCHES) break
    }
    return out
  }

  // 慢路径：逐字符折叠 + 反查映射（back[k] = 折叠串第 k 个字符来自原串的下标）。
  let folded = ''
  const back: number[] = []
  for (let i = 0; i < full.length; i++) {
    const lc = full[i].toLowerCase()
    for (let k = 0; k < lc.length; k++) back.push(i)
    folded += lc
  }
  let from = 0
  let at: number
  while ((at = folded.indexOf(needle, from)) !== -1) {
    const start = back[at]
    const end = at + needle.length < back.length ? back[at + needle.length] : full.length
    out.push({ start, end })
    from = at + needle.length
    if (out.length >= MAX_MATCHES) break
  }
  return out
}

/**
 * 把全局偏移 pos 映射回某文本节点内的 (node, offset)。
 * atEnd 控制边界归属：起点(false)落边界归右侧节点开头；终点(true)落边界归左侧节点末尾，
 * 避免区间末端跨进无关后续块导致 getBoundingClientRect 把多块矩形并起来。
 */
export function locateOffset(
  segs: TextSeg[],
  pos: number,
  atEnd: boolean,
): { node: Text; offset: number } | null {
  if (!segs || !segs.length) return null
  for (let i = 0; i < segs.length; i++) {
    const seg = segs[i]
    const len = seg.node.nodeValue!.length
    const segEnd = seg.start + len
    const hit = atEnd ? pos <= segEnd : pos < segEnd
    if (hit || i === segs.length - 1) {
      return { node: seg.node, offset: clamp(pos - seg.start, 0, len) }
    }
  }
  const last = segs[segs.length - 1]
  return { node: last.node, offset: last.node.nodeValue!.length }
}

/**
 * 遍历 .md-body 内文本节点 → 拼长串 + 段映射，支持跨内联标签匹配。
 * 跳过 KaTeX 隐藏的 MathML 源码副本（.katex-mathml），避免看不见的重复匹配。
 */
export function buildTextIndex(bodyEl: HTMLElement | null): TextIndex {
  if (!bodyEl) return { full: '', segs: [] }
  const walker = document.createTreeWalker(bodyEl, NodeFilter.SHOW_TEXT, {
    acceptNode(node: Node): number {
      if (!node.nodeValue) return NodeFilter.FILTER_REJECT
      const pe = (node as Text).parentElement
      if (pe && pe.closest('.katex-mathml')) return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    },
  })
  let full = ''
  const segs: TextSeg[] = []
  let n: Node | null
  while ((n = walker.nextNode())) {
    const t = n as Text
    segs.push({ node: t, start: full.length })
    full += t.nodeValue
  }
  return { full, segs }
}
