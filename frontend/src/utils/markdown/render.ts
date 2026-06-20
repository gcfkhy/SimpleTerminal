import MarkdownIt from 'markdown-it'
// @ts-ignore 无类型声明
import markdownItCheckbox from 'markdown-it-checkbox'
import markdownItAnchor from 'markdown-it-anchor'
// @ts-ignore 无类型声明
import markdownItToc from 'markdown-it-toc-done-right'
import hljs from 'highlight.js'
import katexPlugin from './ext/markdown-it-katex'
import mermaidPlugin from './ext/markdown-it-mermaid'

export interface RenderOptions { breaks?: boolean }

export function createMarkdownIt(options: RenderOptions = {}): MarkdownIt {
  const md = new MarkdownIt({
    html: true,
    breaks: !!options.breaks,
    highlight(str: string, lang: string): string {
      if (lang && hljs.getLanguage(lang)) {
        try {
          str = hljs.highlight(str, { language: lang }).value
        } catch {
          str = md.utils.escapeHtml(str)
        }
      } else {
        str = md.utils.escapeHtml(str)
      }
      return "<pre class='hljs'><code><div>" + str + '</div></code></pre>'
    },
  })
  md.use(markdownItCheckbox)
    .use(markdownItAnchor)
    .use(markdownItToc)
    .use(katexPlugin)
    .use(mermaidPlugin)
  return md
}

export function renderMarkdownToHtml(text: string, options: RenderOptions = {}): string {
  try {
    return createMarkdownIt(options).render(text || '')
  } catch (error) {
    console.error('renderMarkdownToHtml failed:', error)
    const escaped = String(text ?? '')
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    return '<pre class="md-render-error">' + escaped + '</pre>'
  }
}
