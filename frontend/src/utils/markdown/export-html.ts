// @ts-ignore vite raw 导入
import themesCss from '../../assets/markdown/themes.css?raw'
// @ts-ignore
import contentCss from '../../assets/markdown/content.css?raw'
// @ts-ignore
import katexCss from 'katex/dist/katex.min.css?raw'

/**
 * 用「当前已渲染的预览容器」生成自包含 HTML 字符串。
 * 直接取 rootEl 内 .md-body 的 innerHTML（含 mermaid 渲染出的 SVG），保证导出离线可看。
 */
export function buildExportHtml(rootEl: HTMLElement, themeId: string, title: string): string {
  const body = rootEl.querySelector('.md-body')?.innerHTML ?? ''
  return `<!doctype html>
<html data-theme="${themeId}">
<head>
<meta charset="utf-8">
<title>${title.replace(/</g, '&lt;')}</title>
<style>${katexCss}\n${themesCss}\n${contentCss}
body{margin:0;background:var(--md-bg);}
.md-preview-root{background:var(--md-bg);}
</style>
</head>
<body>
<div class="md-preview-root" data-theme="${themeId}">
<div class="md-body">${body}</div>
</div>
</body>
</html>`
}
