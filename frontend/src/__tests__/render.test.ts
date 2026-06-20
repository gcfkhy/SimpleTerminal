import { describe, it, expect } from 'vitest'
import { renderMarkdownToHtml } from '../utils/markdown/render'

describe('renderMarkdownToHtml', () => {
  it('代码块带 hljs 高亮 class', () => {
    const html = renderMarkdownToHtml('```js\nconst a = 1\n```')
    expect(html).toContain("class='hljs'")
    expect(html).toContain('hljs-keyword')
  })
  it('GFM 表格渲染为 <table>', () => {
    const html = renderMarkdownToHtml('| a | b |\n|---|---|\n| 1 | 2 |')
    expect(html).toContain('<table>')
    expect(html).toContain('<td>1</td>')
  })
  it('任务列表渲染 checkbox', () => {
    const html = renderMarkdownToHtml('- [x] done\n- [ ] todo')
    expect(html).toContain('type="checkbox"')
  })
  it('mermaid 围栏转成 div.mermaid', () => {
    const html = renderMarkdownToHtml('```mermaid\ngraph TD;A-->B;\n```')
    expect(html).toContain('<div class="mermaid">')
  })
  it('行内公式 $..$ 由 KaTeX 渲染', () => {
    const html = renderMarkdownToHtml('质量 $E=mc^2$ 公式')
    expect(html).toContain('katex')
  })
  it('[[toc]] 生成目录容器', () => {
    const html = renderMarkdownToHtml('[[toc]]\n\n# 标题一\n## 标题二')
    expect(html).toContain('class="table-of-contents"')
  })
  it('标题带锚点 id', () => {
    const html = renderMarkdownToHtml('# Hello World')
    expect(html).toMatch(/<h1[^>]*id=/)
  })
  it('渲染异常退化为 md-render-error（非字符串入参不崩）', () => {
    // @ts-expect-error 故意传错类型
    const html = renderMarkdownToHtml(null)
    expect(typeof html).toBe('string')
  })
})
