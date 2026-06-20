/* 移植自 vscode-office：markdown-it 子集 LaTeX → KaTeX 渲染 */
import katex from 'katex'
import type MarkdownIt from 'markdown-it'

function math_inline(state: any, silent: boolean): boolean {
  let start, match, token, pos
  if (state.src[state.pos] !== '$') return false
  start = state.pos + 1
  match = start
  while ((match = state.src.indexOf('$', match)) !== -1) {
    pos = match - 1
    while (state.src[pos] === '\\') pos -= 1
    if (((match - pos) % 2) === 1) break
    match += 1
  }
  if (match === -1) {
    if (!silent) state.pending += '$'
    state.pos = start
    return true
  }
  if (match - start === 0) {
    if (!silent) state.pending += '$$'
    state.pos = start + 1
    return true
  }
  if (!silent) {
    token = state.push('math_inline', 'math', 0)
    token.markup = '$'
    token.content = state.src.slice(start, match)
  }
  state.pos = match + 1
  return true
}

function math_block(state: any, start: number, end: number, silent: boolean): boolean {
  let firstLine, lastLine, next, lastPos, found = false, token
  let pos = state.bMarks[start] + state.tShift[start]
  let max = state.eMarks[start]
  if (pos + 2 > max) return false
  if (state.src.slice(pos, pos + 2) !== '$$') return false
  pos += 2
  firstLine = state.src.slice(pos, max)
  if (silent) return true
  if (firstLine.trim().slice(-2) === '$$') {
    firstLine = firstLine.trim().slice(0, -2)
    found = true
  }
  for (next = start; !found;) {
    next++
    if (next >= end) break
    pos = state.bMarks[next] + state.tShift[next]
    max = state.eMarks[next]
    if (pos < max && state.tShift[next] < state.blkIndent) break
    if (state.src.slice(pos, max).trim().slice(-2) === '$$') {
      lastPos = state.src.slice(0, max).lastIndexOf('$$')
      lastLine = state.src.slice(pos, lastPos)
      found = true
    }
  }
  state.line = next + 1
  token = state.push('math_block', 'math', 0)
  token.block = true
  token.content =
    (firstLine && firstLine.trim() ? firstLine + '\n' : '') +
    state.getLines(start + 1, next, state.tShift[start], true) +
    (lastLine && lastLine.trim() ? lastLine : '')
  token.map = [start, state.line]
  token.markup = '$$'
  return true
}

export default function katexPlugin(md: MarkdownIt): void {
  const options: any = { throwOnError: false, strict: false }
  const katexInline = (latex: string) => {
    options.displayMode = false
    try { return katex.renderToString(latex, options) } catch { return latex }
  }
  const katexBlock = (latex: string) => {
    options.displayMode = true
    try { return '<p>' + katex.renderToString(latex, options) + '</p>' } catch { return latex }
  }
  md.inline.ruler.after('escape', 'math_inline', math_inline)
  md.block.ruler.after('blockquote', 'math_block', math_block, {
    alt: ['paragraph', 'reference', 'blockquote', 'list'],
  })
  md.renderer.rules.math_inline = (tokens, idx) => katexInline(tokens[idx].content)
  md.renderer.rules.math_block = (tokens, idx) => katexBlock(tokens[idx].content) + '\n'
}
