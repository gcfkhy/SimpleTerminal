# Markdown 预览替换为 vscode-office 方案 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 SimpleTerminal 的 Markdown 预览从 `marked`（无高亮、无图表）替换为 vscode-office 的 `markdown-it` 内核，获得代码高亮、Mermaid、KaTeX、任务列表、TOC、18 套主题切换、HTML 导出与单页可选 PDF 导出。

**Architecture:** 原生 Vue 集成——把 vscode-office 的纯 JS 渲染内核（`render.js` + KaTeX/Mermaid 插件 + themes.css）移植进前端 `utils/markdown/`，新增职责单一的 `MarkdownPreview.vue`，由现有 `FilePreview.vue` 在 `.md` 时委托调用。PDF 导出走后端 Go 调用 Windows WebView2 原生 `PrintToPdf`（自定义页高 = 内容高度实现单页不分页），需给 Wails 打 `replace` 补丁取底层指针 + 手写 COM。

**Tech Stack:** Wails v2.12.0 + Vue 3 + TS + Vite 5；markdown-it@14 + 插件；katex@0.16；mermaid@11；highlight.js@11；vitest（新增测试）；Go + go-webview2（手写 COM）。

参考规格：`docs/superpowers/specs/2026-06-20-markdown-preview-vscode-office-port-design.md`

---

## 文件结构

| 文件 | 职责 | 动作 |
|---|---|---|
| `frontend/vitest.config.ts` | 单测配置（jsdom） | 创建 |
| `frontend/src/utils/markdown/render.ts` | `createMarkdownIt` / `renderMarkdownToHtml` 纯函数 | 创建 |
| `frontend/src/utils/markdown/ext/markdown-it-katex.ts` | KaTeX 行内/块级插件（渲染期出 HTML） | 创建（移植） |
| `frontend/src/utils/markdown/ext/markdown-it-mermaid.ts` | 把 ```mermaid 转 `<div class="mermaid">` | 创建（移植） |
| `frontend/src/utils/markdown/themes.ts` | 18 主题 `{id,name,group}` 列表 + 默认 id | 创建（移植） |
| `frontend/src/utils/markdown/export-html.ts` | 生成自包含 HTML 字符串 | 创建 |
| `frontend/src/assets/markdown/themes.css` | 18 主题 CSS 变量（命名空间化到 `.md-preview-root`） | 创建（移植+改写） |
| `frontend/src/assets/markdown/content.css` | `.md-body`/`.hljs` 内容样式（命名空间化） | 创建（移植+改写） |
| `frontend/src/components/MarkdownPreview.vue` | 渲染编排：注入、mermaid 后处理、主题切换、锚点、导出按钮 | 创建 |
| `frontend/src/components/FilePreview.vue` | `.md` 分支改为委托 `MarkdownPreview`；删旧 marked 路径与样式 | 修改 |
| `frontend/src/__tests__/render.test.ts` | render.ts 单测 | 创建 |
| `app.go` | 新增 `SaveExportFile`、`ExportPdf` 绑定 | 修改 |
| `pdfexport_windows.go` | 手写 WebView2 PrintToPdf COM 调用 | 创建（Phase 2） |
| `go.mod` | 启用 `replace` 指向本地 Wails 副本 | 修改（Phase 2） |

**不触及**：`useFileTree`、`App.vue` 布局、终端/PTY、cd-sync、`Divider`、`FileTreeNode` 等。

---

## Phase 1：渲染内核 + 主题 + 组件 + HTML 导出

### Task 0: 安装依赖 + 配置 vitest

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/vitest.config.ts`

- [ ] **Step 1: 安装运行时依赖**

Run（在 `frontend/` 目录）：
```bash
npm i markdown-it@^14.1.0 markdown-it-anchor@^9.2.0 markdown-it-checkbox@^1.1.0 markdown-it-toc-done-right@^4.2.0 katex@^0.16.11 mermaid@^11.4.0
```

- [ ] **Step 2: 安装类型与测试依赖**

Run：
```bash
npm i -D @types/markdown-it@^14.1.2 vitest@^2.1.0 jsdom@^25.0.0
```

- [ ] **Step 3: 加测试脚本**

修改 `frontend/package.json` 的 `scripts`，新增 `"test": "vitest run"` 与 `"test:watch": "vitest"`：
```json
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc --noEmit && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest"
  },
```

- [ ] **Step 4: 创建 vitest 配置**

创建 `frontend/vitest.config.ts`：
```ts
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.ts'],
  },
})
```

- [ ] **Step 5: 验证测试器可运行**

Run：`cd frontend && npx vitest run`
Expected: 退出码 0，提示 “No test files found”（暂无用例）。

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vitest.config.ts
git commit -m "chore: 引入 markdown-it 全家桶 + vitest 测试器"
```

---

### Task 1: 移植渲染内核（render.ts + 两个插件）

**Files:**
- Create: `frontend/src/utils/markdown/ext/markdown-it-katex.ts`
- Create: `frontend/src/utils/markdown/ext/markdown-it-mermaid.ts`
- Create: `frontend/src/utils/markdown/render.ts`
- Test: `frontend/src/__tests__/render.test.ts`

- [ ] **Step 1: 移植 KaTeX 插件**

创建 `frontend/src/utils/markdown/ext/markdown-it-katex.ts`（从 vscode-office `ext/markdown-it-katex.js` 移植；`require('katex')` 改为 import，`module.exports` 改为 `export default`，逻辑不变）：
```ts
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
```

- [ ] **Step 2: 移植 Mermaid 插件**

创建 `frontend/src/utils/markdown/ext/markdown-it-mermaid.ts`。与 vscode-office 相比有意修改：移除 VSCode 专属的 `loadPreferences`；移除 mermaid v11 已变为异步的 `mermaid.parse` 同步调用（避免误判），fence 内容用 `md.utils.escapeHtml` 转义后塞进 `<div class="mermaid">`，真正渲染与报错交给组件里的 `mermaid.run()`：
```ts
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
```

- [ ] **Step 3: 写 render.ts 的失败测试**

创建 `frontend/src/__tests__/render.test.ts`：
```ts
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
```

- [ ] **Step 4: 运行测试确认失败**

Run：`cd frontend && npx vitest run src/__tests__/render.test.ts`
Expected: FAIL，报 `Cannot find module '../utils/markdown/render'`。

- [ ] **Step 5: 实现 render.ts**

创建 `frontend/src/utils/markdown/render.ts`（移植自 `render.js`：ESM 化；`hljs.highlight` 改用 highlight.js 11 新签名 `(str,{language})`，与本项目 `FilePreview.vue` 现有用法一致；去掉 plantuml）：
```ts
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
```

- [ ] **Step 6: 运行测试确认通过**

Run：`cd frontend && npx vitest run src/__tests__/render.test.ts`
Expected: PASS（8 个用例全绿）。若 `[[toc]]` 用例的 class 名不符，按实际输出调整断言（toc-done-right 默认容器类为 `table-of-contents`）。

- [ ] **Step 7: Commit**

```bash
git add frontend/src/utils/markdown frontend/src/__tests__/render.test.ts
git commit -m "feat: 移植 vscode-office markdown-it 渲染内核(含 KaTeX/Mermaid 插件)"
```

---

### Task 2: 移植主题数据与 CSS（命名空间化）

**Files:**
- Create: `frontend/src/utils/markdown/themes.ts`
- Create: `frontend/src/assets/markdown/themes.css`
- Create: `frontend/src/assets/markdown/content.css`

- [ ] **Step 1: 移植主题列表**

创建 `frontend/src/utils/markdown/themes.ts`（即 vscode-office `markdownThemes.ts`，去掉 VSCode 类型导出，保持 18 项不变）：
```ts
export interface MarkdownTheme { id: string; name: string; group: 'light' | 'dark' }
export const DEFAULT_THEME_ID = 'catppuccin-mocha'
export const MARKDOWN_THEMES: MarkdownTheme[] = [
  { id: 'catppuccin-mocha', name: 'Catppuccin Mocha', group: 'dark' },
  { id: 'catppuccin-macchiato', name: 'Catppuccin Macchiato', group: 'dark' },
  { id: 'catppuccin-frappe', name: 'Catppuccin Frappé', group: 'dark' },
  { id: 'dracula', name: 'Dracula', group: 'dark' },
  { id: 'nord', name: 'Nord', group: 'dark' },
  { id: 'one-dark', name: 'One Dark', group: 'dark' },
  { id: 'tokyo-night', name: 'Tokyo Night', group: 'dark' },
  { id: 'gruvbox-dark', name: 'Gruvbox Dark', group: 'dark' },
  { id: 'solarized-dark', name: 'Solarized Dark', group: 'dark' },
  { id: 'rose-pine', name: 'Rosé Pine', group: 'dark' },
  { id: 'github-light', name: 'GitHub Light', group: 'light' },
  { id: 'catppuccin-latte', name: 'Catppuccin Latte', group: 'light' },
  { id: 'solarized-light', name: 'Solarized Light', group: 'light' },
  { id: 'gruvbox-light', name: 'Gruvbox Light', group: 'light' },
  { id: 'one-light', name: 'One Light', group: 'light' },
  { id: 'rose-pine-dawn', name: 'Rosé Pine Dawn', group: 'light' },
  { id: 'ayu-light', name: 'Ayu Light', group: 'light' },
  { id: 'tokyo-night-light', name: 'Tokyo Night Light', group: 'light' },
]
```

- [ ] **Step 2: 移植并命名空间化 themes.css**

把 vscode-office `resource/markdown/themes.css`（全文，18 个 `[data-theme]` 块）复制到 `frontend/src/assets/markdown/themes.css`，然后对**选择器**做机械替换，使变量只在预览容器内生效、不污染全局：
- 文件开头的 `:root,\n[data-theme="catppuccin-mocha"] {` → 改为 `.md-preview-root,\n.md-preview-root[data-theme="catppuccin-mocha"] {`
- 其余每个 `[data-theme="X"] {` → 改为 `.md-preview-root[data-theme="X"] {`

变量内容（`--md-*`、`--hl-*` 全部值）保持原样不变。校验：文件内不应再出现裸 `:root` 或裸 `[data-theme`（无 `.md-preview-root` 前缀）。

- [ ] **Step 3: 移植并命名空间化 content.css**

创建 `frontend/src/assets/markdown/content.css`，内容取自 vscode-office `resource/markdown/preview.css` 的**第 3–57 行**（即 `.md-body*`、`.md-render-error`、`.hljs*` 这段内容样式；**不要**搬第 1–2 行的 `html,body` 和第 59 行起的 `#md-*` 悬浮按钮——前者会改全局背景，后者由 Vue 组件自己实现）。对每条规则前缀 `.md-preview-root `：
```css
/* 移植自 vscode-office preview.css 内容样式，命名空间化到 .md-preview-root */
.md-preview-root .md-body {
  padding: 16px 24px;
  font-family: 'MiSans','Segoe UI',sans-serif;
  font-size: 14px; line-height: 1.7; color: var(--md-fg);
}
.md-preview-root .md-render-error { color: var(--md-code-fg); white-space: pre-wrap; }
.md-preview-root .md-body h1,.md-preview-root .md-body h2,.md-preview-root .md-body h3,
.md-preview-root .md-body h4,.md-preview-root .md-body h5,.md-preview-root .md-body h6 {
  color: var(--md-heading); margin: 1.2em 0 0.4em; font-weight: 600; line-height: 1.3;
}
.md-preview-root .md-body h1 { font-size:1.6em; border-bottom:1px solid var(--md-heading-border); padding-bottom:0.3em; }
.md-preview-root .md-body h2 { font-size:1.3em; border-bottom:1px solid var(--md-heading-border); padding-bottom:0.2em; }
.md-preview-root .md-body h3 { font-size:1.1em; }
.md-preview-root .md-body p { margin: 0.6em 0; }
.md-preview-root .md-body ul,.md-preview-root .md-body ol { padding-left:1.5em; margin:0.5em 0; }
.md-preview-root .md-body li { margin:0.2em 0; }
.md-preview-root .md-body a { color: var(--md-link); text-decoration:none; }
.md-preview-root .md-body a:hover { text-decoration:underline; }
.md-preview-root .md-body code {
  font-family:'SF Mono',Consolas,'MiSans',monospace; font-size:0.88em; font-weight:500;
  background: var(--md-code-bg); color: var(--md-code-fg); padding:0.1em 0.4em; border-radius:4px;
}
.md-preview-root .md-body pre {
  background: var(--md-pre-bg); border:1px solid var(--md-pre-border); border-radius:6px;
  padding:12px; overflow-x:auto; margin:0.8em 0;
}
.md-preview-root .md-body pre code {
  background:transparent; padding:0; font-size:13px; font-weight:500; color: var(--md-fg);
  font-family:'SF Mono',Consolas,'MiSans',monospace;
}
.md-preview-root .md-body blockquote {
  border-left:3px solid var(--md-quote-border); margin:0.8em 0; padding:0.3em 1em;
  color: var(--md-quote-fg); background: var(--md-quote-bg); border-radius:0 4px 4px 0;
}
.md-preview-root .md-body hr { border:none; border-top:1px solid var(--md-border); margin:1em 0; }
.md-preview-root .md-body table { border-collapse:collapse; width:100%; margin:0.8em 0; font-size:13px; }
.md-preview-root .md-body th,.md-preview-root .md-body td { border:1px solid var(--md-border); padding:6px 10px; text-align:left; }
.md-preview-root .md-body th { background: var(--md-table-head-bg); color: var(--md-heading); }
.md-preview-root .md-body tr:nth-child(even) { background: var(--md-table-stripe); }
.md-preview-root .md-body img { max-width:100%; border-radius:4px; }
.md-preview-root .hljs { color: var(--md-fg); background: transparent; }
.md-preview-root .hljs-comment,.md-preview-root .hljs-quote { color: var(--hl-comment); font-style: italic; }
.md-preview-root .hljs-keyword,.md-preview-root .hljs-selector-tag,.md-preview-root .hljs-name,.md-preview-root .hljs-tag { color: var(--hl-keyword); }
.md-preview-root .hljs-string,.md-preview-root .hljs-section,.md-preview-root .hljs-addition { color: var(--hl-string); }
.md-preview-root .hljs-number,.md-preview-root .hljs-literal,.md-preview-root .hljs-bullet,.md-preview-root .hljs-link,.md-preview-root .hljs-deletion { color: var(--hl-number); }
.md-preview-root .hljs-title,.md-preview-root .hljs-title.function_,.md-preview-root .hljs-function .hljs-title { color: var(--hl-function); }
.md-preview-root .hljs-attr,.md-preview-root .hljs-variable,.md-preview-root .hljs-template-variable,.md-preview-root .hljs-params,.md-preview-root .hljs-property { color: var(--hl-attr); }
.md-preview-root .hljs-built_in,.md-preview-root .hljs-type,.md-preview-root .hljs-title.class_,.md-preview-root .hljs-class .hljs-title { color: var(--hl-class); }
.md-preview-root .hljs-meta { color: var(--hl-meta); }
.md-preview-root .hljs-symbol { color: var(--hl-symbol); }
.md-preview-root .hljs-regexp { color: var(--hl-regexp); }
.md-preview-root .hljs-selector-id,.md-preview-root .hljs-selector-class,.md-preview-root .hljs-selector-attr,.md-preview-root .hljs-selector-pseudo { color: var(--hl-class); }
.md-preview-root .hljs-doctag,.md-preview-root .hljs-strong { font-weight:bold; }
.md-preview-root .hljs-emphasis { font-style:italic; }
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/utils/markdown/themes.ts frontend/src/assets/markdown
git commit -m "feat: 移植 18 套 markdown 主题与内容样式(命名空间化到 .md-preview-root)"
```

---

### Task 3: 新建 MarkdownPreview.vue

**Files:**
- Create: `frontend/src/components/MarkdownPreview.vue`

- [ ] **Step 1: 创建组件**

创建 `frontend/src/components/MarkdownPreview.vue`：
```vue
<script setup lang="ts">
import { ref, watch, computed, onMounted, nextTick } from 'vue'
import mermaid from 'mermaid'
import 'katex/dist/katex.min.css'
import '../assets/markdown/themes.css'
import '../assets/markdown/content.css'
import { renderMarkdownToHtml } from '../utils/markdown/render'
import { MARKDOWN_THEMES, DEFAULT_THEME_ID } from '../utils/markdown/themes'
import { BrowserOpenURL } from '../../wailsjs/runtime'

const props = defineProps<{ source: string; filePath: string }>()

const theme = ref(localStorage.getItem('md-preview-theme') || DEFAULT_THEME_ID)
const themeOpen = ref(false)
const html = computed(() => renderMarkdownToHtml(props.source))
const bodyEl = ref<HTMLElement | null>(null)
const rootEl = ref<HTMLElement | null>(null)

const isDark = computed(
  () => MARKDOWN_THEMES.find(t => t.id === theme.value)?.group !== 'light',
)
const darkThemes = MARKDOWN_THEMES.filter(t => t.group === 'dark')
const lightThemes = MARKDOWN_THEMES.filter(t => t.group === 'light')

function pickTheme(id: string) {
  theme.value = id
  localStorage.setItem('md-preview-theme', id)
  themeOpen.value = false
}

// mermaid：每次内容/主题变化后，对容器内 .mermaid 节点重渲
async function runMermaid() {
  if (!bodyEl.value) return
  const nodes = bodyEl.value.querySelectorAll<HTMLElement>('.mermaid:not([data-processed])')
  if (!nodes.length) return
  mermaid.initialize({
    startOnLoad: false,
    theme: isDark.value ? 'dark' : 'default',
    securityLevel: 'loose',
  })
  try {
    await mermaid.run({ nodes: Array.from(nodes) })
  } catch (e) {
    console.error('mermaid render error:', e)
  }
}

watch([html, theme], async () => {
  await nextTick()
  await runMermaid()
})
onMounted(async () => {
  await nextTick()
  await runMermaid()
})

// 链接：外链走系统浏览器；页内锚点容器内平滑滚动
function onClick(e: MouseEvent) {
  const a = (e.target as HTMLElement).closest('a')
  if (!a) return
  const href = a.getAttribute('href')
  if (!href) return
  if (href.startsWith('#')) {
    e.preventDefault()
    const target = rootEl.value?.querySelector(decodeURIComponent(href))
    target?.scrollIntoView({ behavior: 'smooth' })
    return
  }
  e.preventDefault()
  BrowserOpenURL(href)
}

defineExpose({ rootEl, bodyEl, theme })
</script>

<template>
  <div class="md-preview-root" :data-theme="theme" ref="rootEl">
    <div class="md-body" ref="bodyEl" v-html="html" @click="onClick"></div>

    <div class="md-toolbar">
      <button class="md-tool-btn" title="主题" @click="themeOpen = !themeOpen">🎨</button>
    </div>
    <div v-if="themeOpen" class="md-theme-panel">
      <div class="md-theme-group">暗色</div>
      <div v-for="t in darkThemes" :key="t.id"
           class="md-theme-item" :class="{ active: t.id === theme }"
           @click="pickTheme(t.id)">{{ t.name }}</div>
      <div class="md-theme-group">亮色</div>
      <div v-for="t in lightThemes" :key="t.id"
           class="md-theme-item" :class="{ active: t.id === theme }"
           @click="pickTheme(t.id)">{{ t.name }}</div>
    </div>
  </div>
</template>

<style scoped>
.md-preview-root {
  position: relative;
  height: 100%;
  overflow: auto;
  background: var(--md-bg);
}
.md-toolbar {
  position: absolute; right: 14px; bottom: 14px; display: flex; gap: 8px; z-index: 10;
}
.md-tool-btn {
  width: 34px; height: 34px; border-radius: 50%;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); color: var(--md-fg);
  font-size: 16px; cursor: pointer; opacity: 0.85;
}
.md-tool-btn:hover { opacity: 1; }
.md-theme-panel {
  position: absolute; right: 14px; bottom: 56px; max-height: 60vh; overflow: auto;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); border-radius: 8px;
  padding: 6px; z-index: 11; min-width: 170px; box-shadow: 0 6px 24px rgba(0,0,0,0.35);
}
.md-theme-group { font-size: 11px; color: var(--md-muted); margin: 6px 6px 2px; }
.md-theme-item { padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 13px; color: var(--md-fg); white-space: nowrap; }
.md-theme-item:hover { background: var(--md-pre-bg); }
.md-theme-item.active { background: var(--md-pre-bg); font-weight: 600; }
.md-theme-item.active::after { content: ' ✓'; color: var(--md-link); }
</style>
```

> 注：`v-html` + scoped 样式作用不到注入内容，因此正文样式靠 `themes.css`/`content.css`（全局、命名空间化），组件 scoped 样式只管容器与工具条。

- [ ] **Step 2: 构建校验（类型 + 打包）**

Run：`cd frontend && npx vue-tsc --noEmit`
Expected: 无类型错误（若 `markdown-it-checkbox`/`toc-done-right` 报缺类型，确认 render.ts 已用 `// @ts-ignore`）。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/MarkdownPreview.vue
git commit -m "feat: 新增 MarkdownPreview 组件(渲染+主题切换+mermaid 后处理+锚点)"
```

---

### Task 4: FilePreview 委托 + 清理旧 marked 路径

**Files:**
- Modify: `frontend/src/components/FilePreview.vue`

- [ ] **Step 1: 改 script —— 用 MarkdownPreview 替换 marked**

在 `FilePreview.vue`：
1. 删除第 6 行 `import { marked } from 'marked'`，新增 `import MarkdownPreview from './MarkdownPreview.vue'`。
2. 删除 `markdownHtml` ref（第 120 行）。
3. 新增 `const markdownRaw = ref('')`。
4. watch 中 markdown 分支（第 145–147 行）改为只取原文：
```ts
    } else if (kind.value === 'markdown') {
      markdownRaw.value = await ReadFileText(file.path, 500 * 1024)
```
5. watch 顶部重置区把 `markdownHtml.value = ''` 改为 `markdownRaw.value = ''`。

- [ ] **Step 2: 改 template —— markdown 分支换成组件**

把第 197–199 行：
```vue
      <div v-else-if="kind === 'markdown'" class="md-wrap">
        <div class="md-body" v-html="markdownHtml"></div>
      </div>
```
替换为：
```vue
      <div v-else-if="kind === 'markdown'" class="md-wrap">
        <MarkdownPreview :source="markdownRaw" :file-path="file.path" />
      </div>
```

- [ ] **Step 3: 删除失效的旧 markdown 样式**

删除 `FilePreview.vue` 中第 306–406 行的 `/* Markdown 渲染区 */` 整段（`.md-wrap`、`.md-body` 及其所有 `:deep(...)` 子规则）——这些是旧 marked 路径的样式，现在由 MarkdownPreview 自带主题样式取代。仅保留一条让组件撑满的容器规则，新增：
```css
.md-wrap { height: 100%; }
```

- [ ] **Step 4: 跑起来手动验证渲染与主题**

Run：`wails dev`（项目根目录）。在文件树点开一个含代码块/表格/任务列表/Mermaid/公式的 `.md`，逐项确认：
- 代码块有语法高亮（彩色 token）
- 表格、任务列表 checkbox 正常
- ```mermaid 出图、`$E=mc^2$` 出公式
- 点 🎨 切几个主题，背景/配色随之变，且**终端和文件树不受影响**
- 外链点击走系统浏览器；`[[toc]]` 目录里的锚点点击能滚动
Expected: 全部符合；控制台无红色报错。

- [ ] **Step 5: 跑单测确保无回归**

Run：`cd frontend && npx vitest run`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/FilePreview.vue
git commit -m "feat: FilePreview 的 .md 预览改用 MarkdownPreview，移除旧 marked 路径"
```

---

### Task 5: HTML 导出（自包含单文件）

**Files:**
- Create: `frontend/src/utils/markdown/export-html.ts`
- Modify: `app.go`
- Modify: `frontend/src/components/MarkdownPreview.vue`

- [ ] **Step 1: 后端新增 SaveExportFile**

在 `app.go` 末尾新增（`os` 已导入）：
```go
// SaveExportFile 弹出保存对话框并把 content 写入所选路径，返回路径（取消时空串）。
func (a *App) SaveExportFile(defaultName, content string) (string, error) {
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "导出",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}
```

- [ ] **Step 2: 重新生成 Wails 绑定**

Run：`wails generate module`（项目根目录）。
Expected: `frontend/wailsjs/go/main/App.d.ts` 出现 `SaveExportFile(arg1:string,arg2:string):Promise<string>`。

- [ ] **Step 3: 写 export-html.ts**

创建 `frontend/src/utils/markdown/export-html.ts`。用 Vite 的 `?raw` 把三份 CSS 内联进单文件；Mermaid 用传入的「已渲染 DOM」里的 SVG，使导出文件离线静态可看：
```ts
// @ts-ignore vite raw 导入
import themesCss from '../../assets/markdown/themes.css?raw'
// @ts-ignore
import contentCss from '../../assets/markdown/content.css?raw'
// @ts-ignore
import katexCss from 'katex/dist/katex.min.css?raw'

/**
 * 用「当前已渲染的预览容器」生成自包含 HTML 字符串。
 * 直接取 rootEl.innerHTML（含 mermaid 渲染出的 SVG），保证导出离线可看。
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
```

- [ ] **Step 4: MarkdownPreview 接入导出按钮**

在 `MarkdownPreview.vue` 中：
1. import：`import { buildExportHtml } from '../utils/markdown/export-html'` 和 `import { SaveExportFile } from '../../wailsjs/go/main/App'`。
2. 新增方法：
```ts
async function exportHtml() {
  if (!rootEl.value) return
  const name = (props.filePath.split(/[\\/]/).pop() || 'export').replace(/\.md$/i, '') + '.html'
  const content = buildExportHtml(rootEl.value, theme.value, name)
  try {
    const saved = await SaveExportFile(name, content)
    if (saved) console.log('导出成功:', saved)
  } catch (e) {
    console.error('导出失败:', e)
  }
}
```
3. 工具条加按钮（在 🎨 按钮前）：
```vue
      <button class="md-tool-btn" title="导出 HTML" @click="exportHtml">⬇</button>
```

- [ ] **Step 5: 手动验证 HTML 导出**

Run：`wails dev` → 打开 md → 点 ⬇ → 保存 → 用浏览器打开导出文件。
Expected: 离线可看，样式/代码高亮/Mermaid SVG/公式都在，主题与导出时一致。

- [ ] **Step 6: Commit**

```bash
git add app.go frontend/wailsjs frontend/src/utils/markdown/export-html.ts frontend/src/components/MarkdownPreview.vue
git commit -m "feat: Markdown 预览支持导出自包含 HTML"
```

---

## Phase 2：单页可选 PDF 导出（WebView2 PrintToPdf）

> ⚠️ 本阶段涉及给 Wails 打 `replace` 补丁取底层 WebView2 指针 + 手写 COM，外部 SDK 细节须在 spike 中钉死。**先做 Task 6 spike，通过后再做 Task 7。**

### Task 6: PDF 技术验证 spike（一次性，可丢弃）

**目标**：在动正式代码前，最小化验证三件事，把不确定性清零。

- [ ] **Step 1: 定位并准备本地 Wails 副本**

确认本机 Go module 缓存路径（当前用户 `gcfkh`）：
```bash
go env GOMODCACHE
```
把 `wailsapp/wails/v2@v2.12.0` 复制到一个可写目录（如 `E:\wails-local`），并在 `go.mod` 第 41 行启用 replace：
```
replace github.com/wailsapp/wails/v2 v2.12.0 => E:\wails-local\wails\v2
```
Run：`go build ./...`（项目根目录）。
Expected: 用本地副本构建成功，应用行为不变。

- [ ] **Step 2: 在 Wails Windows frontend 暴露 chromium 指针**

在本地 Wails 副本 `v2/internal/frontend/desktop/windows/frontend.go` 里，给 `Frontend` 加一个导出方法返回内部 `*edge.Chromium`（字段名以源码为准，通常为 `f.chromium`）：
```go
// PrintChromium 暴露底层 WebView2，供 PDF 导出（自定义补丁）。
func (f *Frontend) PrintChromium() *edge.Chromium { return f.chromium }
```
并查明 `application.go` / `runtime` 如何从 app 拿到该 Frontend 实例（spike 输出：记录确切取用链路）。
Expected: 能在项目 Go 代码里拿到 `*edge.Chromium` 非空指针。

- [ ] **Step 3: 验证 PrintToPdf 可调 + 打印 CSS 隔离 + 单页**

写一段一次性试验代码（spike，不求优雅）：用 `chromium.GetController().GetCoreWebView2()` 拿 `*ICoreWebView2`，`QueryInterface` 到 `ICoreWebView2_7`，手写最小 `PrintToPdf` 调用，传一个自定义 `ICoreWebView2PrintSettings`（PageWidth=8.27in、PageHeight= 一个大值、ShouldPrintBackgrounds=TRUE）。前端临时给预览容器加 `md-print-isolate` 类并注入打印 CSS（见 Task 7 Step 3）。导出一份 PDF 并人工检查：
- ① 文件能生成、能打开；
- ② **只含 markdown**（无终端/文件树）；
- ③ 文字可选可复制；
- ④ PageHeight 大值时是否单页、>200 英寸时报错还是截断（记录真实行为）。

IID 取自 WebView2 SDK 头文件 `WebView2.h`：`ICoreWebView2_7`、`ICoreWebView2PrintSettings`、`ICoreWebView2PrintToPdfCompletedHandler` 的 `IID_*`。vtable 写法照抄本地 go-webview2 `pkg/edge/ICoreWebView2Controller.go` 的 `ComProc` 模式。
Expected: 四点全部得到明确结论；spike 代码与结论记录在提交信息或 `docs/superpowers/plans/` 旁注。

- [ ] **Step 4: 记录 spike 结论**

把以下结论写进本计划文件末尾「Spike 结论」小节：取 chromium 的确切链路、三个 IID 的 GUID、PrintToPdf 参数顺序、>200 英寸的真实表现。Task 7 据此实现。

> 若 Step 2 的 replace 改源过于脆弱，spike 的备选结论可改为「自建离屏 WebView2」路线（go-webview2 公共 API 自建 chromium + 隐藏窗口 + STA 消息泵），在结论里说明选型。

---

### Task 7: PDF 导出正式实现

**Files:**
- Create: `pdfexport_windows.go`
- Modify: `app.go`
- Modify: `frontend/src/components/MarkdownPreview.vue`
- Modify: `go.mod`（保留 Task 6 的 replace）

> 本任务的 COM vtable 具体字段以 Task 6 spike 结论为准；以下给出**确定可写**的外围代码（Go 绑定签名、打印 CSS、前端测高与流程），COM 调用按 spike 钉死的 IID/参数填入 `pdfexport_windows.go`。

- [ ] **Step 1: pdfexport COM 封装骨架**

创建 `pdfexport_windows.go`，对外只暴露一个函数（内部填入 spike 验证过的 COM 调用）：
```go
//go:build windows

package main

// PrintToPdf 用底层 WebView2 把当前文档打印为 PDF 到 savePath。
// pageW/pageH 单位英寸；scale ∈ [0.1,2.0]；printBg 是否打印背景色。
// chromium 由 Wails 本地补丁的 PrintChromium() 提供。
func PrintToPdf(savePath string, pageW, pageH, scale float64, printBg bool) error {
	// 1) c := <从 Wails Frontend 拿 *edge.Chromium>（spike Step 2 链路）
	// 2) core := c.GetController().GetCoreWebView2()
	// 3) cv7 := QueryInterface(core, IID_ICoreWebView2_7)
	// 4) settings := cv7.CreatePrintSettings(); 设 PageWidth/PageHeight/ScaleFactor/ShouldPrintBackgrounds
	// 5) done := make(chan error, 1)
	//    cv7.PrintToPdf(savePath, settings, handler{done})  // handler 回调写入 done
	// 6) return <-done
	panic("fill from spike Task 6 conclusions")
}
```

- [ ] **Step 2: app.go 新增 ExportPdf 绑定**

在 `app.go` 新增（内部用 SaveFileDialog 取路径，再调 `PrintToPdf`）：
```go
// ExportPdf 把当前预览导出为单页 PDF。pageHeightIn 由前端按内容高度算好；
// scale<1 表示触发了超长缩放。返回保存路径（取消时空串）。
func (a *App) ExportPdf(defaultName string, pageWidthIn, pageHeightIn, scale float64) (string, error) {
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "导出 PDF",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := PrintToPdf(path, pageWidthIn, pageHeightIn, scale, true); err != nil {
		return "", err
	}
	return path, nil
}
```
Run：`wails generate module` 重新生成绑定。
Expected: `App.d.ts` 出现 `ExportPdf`。

- [ ] **Step 3: 打印隔离 CSS（全局）**

在 `frontend/src/style.css` 末尾追加（仅打印时生效：隐藏除带 `md-print-isolate` 标记外的一切，并解除预览高度约束让全文展开）：
```css
@media print {
  body * { visibility: hidden !important; }
  .md-print-isolate, .md-print-isolate * { visibility: visible !important; }
  .md-print-isolate {
    position: absolute !important; left: 0; top: 0; width: 100%;
    height: auto !important; max-height: none !important; overflow: visible !important;
  }
  .md-print-isolate .md-toolbar, .md-print-isolate .md-theme-panel { display: none !important; }
}
```

- [ ] **Step 4: 前端测高 + 导出流程**

在 `MarkdownPreview.vue` 加 PDF 导出方法（A4 宽 8.27in；高度按内容 `scrollHeight/96` 英寸算；>195in 自动降缩放并提示）：
```ts
import { ExportPdf } from '../../wailsjs/go/main/App'

const toast = ref('')
function showToast(msg: string) { toast.value = msg; setTimeout(() => (toast.value = ''), 2600) }

async function exportPdf() {
  const root = rootEl.value
  if (!root) return
  const PAGE_W = 8.27
  const MAX_H = 195 // 英寸，留余量避开 200 硬上限
  root.classList.add('md-print-isolate')
  await nextTick()
  const heightIn = root.scrollHeight / 96
  let scale = 1
  let pageH = heightIn
  if (heightIn > MAX_H) {
    scale = Math.max(0.1, MAX_H / heightIn)
    pageH = heightIn * scale
    showToast(`内容过长，已自动缩放至 ${Math.round(scale * 100)}%`)
  }
  const name = (props.filePath.split(/[\\/]/).pop() || 'export').replace(/\.md$/i, '') + '.pdf'
  try {
    const saved = await ExportPdf(name, PAGE_W, pageH, scale)
    if (saved) showToast('PDF 已导出')
  } catch (e) {
    showToast('PDF 导出失败')
    console.error(e)
  } finally {
    root.classList.remove('md-print-isolate')
  }
}
```
工具条加按钮 + toast 元素：
```vue
      <button class="md-tool-btn" title="导出 PDF" @click="exportPdf">📄</button>
```
```vue
    <div v-if="toast" class="md-toast">{{ toast }}</div>
```
scoped 样式加：
```css
.md-toast {
  position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  padding: 10px 20px; border-radius: 10px; font-size: 14px;
  background: var(--md-ui-bg); color: var(--md-fg); border: 1px solid var(--md-ui-border);
  box-shadow: 0 6px 24px rgba(0,0,0,0.35); z-index: 20;
}
```

- [ ] **Step 5: 手动验证 PDF 导出**

Run：`wails dev` → 打开 md → 点 📄 → 保存。检查导出的 PDF：
- 单页不分页；文字可选可复制（在 PDF 阅读器里框选）；
- 只含 markdown（无终端/文件树）；保留主题底色；
- 找一篇超长 md（>18 屏）导出，确认触发「已自动缩放至 N%」提示且仍单页。
Expected: 全部符合。

- [ ] **Step 6: Commit**

```bash
git add pdfexport_windows.go app.go go.mod frontend/wailsjs frontend/src/style.css frontend/src/components/MarkdownPreview.vue
git commit -m "feat: Markdown 预览支持导出单页可选文字 PDF(WebView2 PrintToPdf)"
```

---

### Task 8: 全量集成验证与收尾

**Files:** 无新增（验证 + 文档）

- [ ] **Step 1: 全量单测 + 构建**

Run：
```bash
cd frontend && npx vitest run && npx vue-tsc --noEmit
```
然后项目根目录 `wails build`。
Expected: 测试全绿、类型无误、构建出 exe。

- [ ] **Step 2: 回归与验收清单（手动）**

逐条对照规格 §13 验收标准走查：渲染品质、18 主题切换且不污染、HTML 导出、PDF 单页可选、超长缩放提示、cd-sync/终端/文件树无回归（切目录、开多 tab、点其它文件类型预览均正常）。

- [ ] **Step 3: 记录 Wails 补丁点**

在 `docs/superpowers/plans/` 旁或 README 注明：本项目对 Wails v2.12.0 打了本地 replace 补丁（暴露 `PrintChromium`），升级 Wails 时需重新施加。

- [ ] **Step 4: Commit（如有文档改动）**

```bash
git add -A
git commit -m "docs: 记录 Markdown 预览迁移完成与 Wails 本地补丁说明"
```

---

## 自审记录（规格覆盖对照）

- 渲染引擎换 markdown-it + 插件 → Task 1 ✅
- 代码高亮 / 表格 / 任务列表 / TOC / 锚点 → Task 1（render）+ Task 2（样式）✅
- Mermaid / KaTeX → Task 1（插件）+ Task 3（mermaid.run / katex css）✅
- 18 主题切换 + 记忆 + 仅作用预览容器 → Task 2 + Task 3 ✅
- HTML 自包含导出 → Task 5 ✅
- PDF 单页 + 文字可选 + 仅含 markdown + 主题底色 → Task 6 spike + Task 7 ✅
- 超长自动降缩放 + 提示 → Task 7 Step 4 ✅
- PlantUML 排除 / 不打包第二个 Chromium → 计划未引入 ✅
- cd-sync/终端/文件树无回归 → Task 4 Step 4、Task 8 Step 2 ✅

**已知风险（见规格 §12）**：Wails 取指针路径、COM IID/vtable、>200in 行为——全部前置到 Task 6 spike 钉死，未通过不进 Task 7。

## Spike 结论
（Task 6 完成后在此填写：取 chromium 链路 / 三个 IID GUID / PrintToPdf 参数顺序 / >200 英寸真实表现 / 选型 = 复用 app WebView2 还是离屏 WebView2）
