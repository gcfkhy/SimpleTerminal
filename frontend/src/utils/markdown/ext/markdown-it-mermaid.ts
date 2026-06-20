import type MarkdownIt from 'markdown-it'

export default function mermaidPlugin(md: MarkdownIt): void {
  const fallback = md.renderer.rules.fence!.bind(md.renderer.rules)
  md.renderer.rules.fence = (tokens, idx, options, env, slf) => {
    const token = tokens[idx]
    const code = token.content.trim()
    const isMermaid =
      token.info === 'mermaid' ||
      ['gantt', 'sequenceDiagram'].includes(code.split(/\n/)[0].trim()) ||
      /^graph (?:TB|BT|RL|LR|TD);?$/.test(code.split(/\n/)[0].trim())
    if (isMermaid) {
      return `<div class="mermaid">${md.utils.escapeHtml(code)}</div>`
    }
    return fallback(tokens, idx, options, env, slf)
  }
}
