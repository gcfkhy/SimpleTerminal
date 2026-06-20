# 设计文档：将 SimpleTerminal 的 Markdown 预览替换为 vscode-office 方案

- 日期：2026-06-20
- 状态：待用户评审
- 目标项目：`E:\资料\Project\SimpleTerminal`（Wails v2.12.0 + Vue 3 + TS，仅 Windows，WebView2 内核）
- 来源项目：`E:\资料\Project\vscode-office`（VSCode 扩展，markdown-it 渲染内核）

## 1. 目标与动机

当前预览基于 `marked@18`（无任何配置），存在明显短板：**Markdown 内代码块不做语法高亮**、无 Mermaid、无数学公式、无 TOC/任务列表、主题固定单一。用户要求换成 vscode-office 的高质量预览。

好消息：vscode-office 的渲染内核 `render.js` 是**纯 JS、零 VSCode 依赖**，可直接移植进 Vue 前端。与 VSCode 耦合的只是 webview 外壳和 puppeteer 导出，这些不移植。

### 范围（已与用户确认）

完整档、一期全部做完：

- 渲染引擎换成 `markdown-it@14` + 插件集
- 代码高亮（修复当前最大缺陷）
- Mermaid 图、KaTeX 数学公式
- 任务列表 checkbox、TOC 目录、锚点跳转
- 18 套主题切换器
- HTML 导出（自包含单文件）
- **PDF 导出：单页不分页 + 文字可选可复制**

### 明确排除

- **PlantUML**：`markdown-it-plantuml` 需把图编码后请求外部 plantuml 服务器渲染，离线桌面 app 不合适。
- **PNG/DOCX 导出**：本期不做。
- **非 Windows 平台**：项目本就仅 Windows（ConPTY/PowerShell 专属）。

## 2. 现状（基线）

| 维度 | 当前 SimpleTerminal |
|---|---|
| 渲染 | `marked(raw, { async:true })`，无 renderer 配置 → 代码块不高亮 |
| 入口组件 | `frontend/src/components/FilePreview.vue`（同时处理 md/code/image/video/text） |
| 数据流 | 文件树点击 → `useFileTree.open()` → `selectedFile` → `FilePreview` watch → 后端 `ReadFileText` → `marked()` → `v-html` |
| 代码文件高亮 | `highlight.js`，样式 `atom-one-dark.min.css`（`main.ts` 导入），仅用于**代码文件**预览，不作用于 md 内代码块 |
| 主题 | 固定 Catppuccin Mocha 暗色（CSS 变量） |
| 外链处理 | `onLinkClick` 拦截 → `BrowserOpenURL` 用系统浏览器打开（`FilePreview.vue`） |
| 后端 | `app.go` 暴露 8 个方法；有 `OpenDirectoryDialog`，**无 SaveFileDialog**，无写文件能力 |
| 关键版本 | `wails/v2 v2.12.0`、`go-webview2 v1.0.22`（间接） |

vscode-office 侧关键资产：`src/service/markdown/render.js`（纯函数 `renderMarkdownToHtml`）、`ext/markdown-it-katex.js`、`ext/markdown-it-mermaid.js`、`resource/markdown/themes.css`（18 主题，含 `--md-*` 与代码高亮 `--hl-*` 变量）、`resource/markdown/preview.css`、`src/provider/markdownThemes.ts`（主题列表）。

## 3. 总体架构

采用**原生 Vue 集成**（不用 iframe）：把 vscode-office 的渲染内核作为前端模块引入，新增一个职责单一的 `MarkdownPreview.vue`，由现有 `FilePreview.vue` 在遇到 `.md` 时委托调用。PDF 导出走后端 Go 调用 WebView2 原生 `PrintToPdf`。

```
frontend/src/
  utils/markdown/
    render.ts                  # 移植 render.js（CJS→ESM）：createMarkdownIt + renderMarkdownToHtml
    highlight.ts               # markdown-it 的 highlight 回调，封装 highlight.js
    ext/markdown-it-katex.ts   # 移植：渲染期把 $..$/$$..$$ 转成 KaTeX HTML
    ext/markdown-it-mermaid.ts # 移植：把 ```mermaid 转成 <div class="mermaid">…</div>
    themes.ts                  # 移植 markdownThemes.ts：18 套主题的 {id,label,dark} 列表
    export-html.ts             # 生成自包含 HTML 字符串（内联 CSS + 渲染后 body）
  components/
    MarkdownPreview.vue        # 新组件（见 §4）
    FilePreview.vue            # 改：.md 分支委托给 MarkdownPreview，其余分支不动
  assets/markdown/
    themes.css                 # 搬运，所有选择器命名空间化到 .md-preview-root（见 §5）
    preview.css                # 搬运并命名空间化

app.go / 新增 Go 文件          # SaveFileDialog 绑定 + PDF 导出（见 §7）
```

**未触及**：cd-sync 架构、文件树、终端 PTY、`useFileTree`、布局/分隔条等一律不动。

### 数据流（入口不变）

```
文件树点击 .md → useFileTree.open() → selectedFile
  → FilePreview watch → ReadFileText(path) 拿到原文 raw
  → 判定 .md → <MarkdownPreview :source="raw" :file-path="path" />
  → render.ts 渲染成 HTML → v-html 注入 .md-preview-root[data-theme]
  → 注入后跑 mermaid.run() 画图（KaTeX 在渲染期已出 HTML，无需后处理）
```

## 4. MarkdownPreview.vue 组件设计

**Props**
- `source: string` — markdown 原文
- `filePath: string` — 当前文件路径（用于相对资源/导出默认名/外链基准）

**Emits**：无（自包含；外链沿用 `BrowserOpenURL`）

**内部职责**
1. **渲染**：`html = renderMarkdownToHtml(source)`；`v-html` 注入到 `<div class="md-preview-root" :data-theme="theme">`。
2. **Mermaid 后处理**：`watch` html 变化，`nextTick` 后对容器内 `.mermaid` 节点调用 `mermaid.run({ nodes })`；失败的图就地降级为错误块（不抛整页）。主题切换时按新主题重渲。
3. **主题切换器**：预览区右上角浮动工具条上一个 🎨 按钮，弹出 18 主题列表；选择写入 `data-theme` 并持久化到 `localStorage('md-preview-theme')`；默认 `catppuccin-mocha`（贴合 app 暗色）。Mermaid 主题随之映射（dark→`dark`，light→`default`）。
4. **TOC / 锚点**：`markdown-it-anchor` 生成 id，`markdown-it-toc-done-right` 支持 `[[toc]]`；点击页内锚点平滑滚动到容器内目标（不污染浏览器历史）。
5. **链接处理**：外链（http/https）→ `BrowserOpenURL`（沿用现状）；页内 `#anchor` → 容器内滚动。
6. **导出工具条**：两个按钮——「导出 HTML」「导出 PDF」（见 §6/§7）。
7. **缩放**（可选，沿用 vscode-office 体验）：Ctrl+滚轮调 `font-size`，持久化。本期可做也可留空，不阻塞主体。

**职责边界**：渲染与导出逻辑放在 `utils/markdown/*`，组件只做编排与 DOM 后处理，保证可单独理解与测试。

## 5. 样式与主题（关键：命名空间隔离）

vscode-office 的 `themes.css` 通过 `[data-theme="…"]` 定义 `--md-*`（正文配色）和 `--hl-*`（代码高亮 token 配色），并用 `.md-body`、`.hljs`、`pre` 等选择器套用。直接全局引入会**污染 app 其它部分**，尤其会和现有代码文件预览用的 `atom-one-dark.min.css` 冲突。

**做法**：移植时把 `themes.css` / `preview.css` 的所有选择器**命名空间化到 `.md-preview-root` 下**：
- `[data-theme="x"]` → `.md-preview-root[data-theme="x"]`
- `.md-body`、`.hljs`、`pre code` 等 → `.md-preview-root .md-body` …

这样：markdown 预览用 themes.css 的高亮配色；代码文件预览继续用 `atom-one-dark`；互不干扰，`main.ts` 的现有导入保留。

KaTeX 样式：`import 'katex/dist/katex.min.css'`（Vite 会一并打包其 woff2 字体）。该 CSS 作用域受 `.katex` class 限制，风险低。

## 6. 导出 HTML

纯前端可完成：
1. `export-html.ts` 生成自包含 HTML 字符串：`<!doctype html><html data-theme="<当前主题>"><head><meta charset><style>{themes.css}{preview.css}{katex.css}</style></head><body><div class="md-preview-root" data-theme="…">{body}</div></body></html>`。
2. **Mermaid 处理**：导出时从**当前 DOM 抓已渲染的 `<svg>`** 内联进 body，使导出文件静态、无需 JS、离线可看。
3. CSS 通过 Vite `?raw` 导入或构建期内联，避免运行时再 fetch。
4. 落盘：调用新增后端 `SaveExportFile(defaultName, content)`（见 §7.1）。

## 7. 后端：保存对话框 + PDF 导出

### 7.1 SaveFileDialog 绑定（低风险）

`app.go` 新增：

```go
func (a *App) SaveExportFile(defaultName, content string) (string, error) {
    path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
        DefaultFilename: defaultName,
        Filters: []wruntime.FileFilter{{DisplayName: "...", Pattern: "*.html;*.pdf"}},
    })
    if err != nil || path == "" { return "", err }
    return path, os.WriteFile(path, []byte(content), 0644)
}
```

HTML 导出复用 `SaveExportFile`（取路径并写入内容）。PDF 导出**不走** `SaveExportFile`，而是在 `ExportPdf` 内部直接调 `wruntime.SaveFileDialog` 仅取路径，文件内容由 WebView2 `PrintToPdf` 直接写盘。

### 7.2 PDF 导出：WebView2 原生 PrintToPdf（高风险核心）

**唯一能同时满足「单页不分页 + 文字可选 + 不再多打包一个 Chromium」的方案**。要点与已查证结论：

- `PrintToPdf` 用 Chromium 自带打印引擎，产出**真矢量、可选可搜**的 PDF（项目输出为 HTML 文字，全部矢量化）。
- `ICoreWebView2PrintSettings` 支持自定义 `PageWidth`/`PageHeight`（**单位英寸**，须 >0）、`ScaleFactor`（0.1~2.0）、`ShouldPrintBackgrounds`（要保留主题底色须设 TRUE）、四边 `Margin`。
- 把 `PageHeight = 内容总高度` → 单页不分页。**硬上限 200 英寸**（PDF 格式 14400 units 限制）。
- 所需最低 WebView2 Runtime：`PrintToPdf` 属 `ICoreWebView2_7`（98.0.1108.43，2022），2026 年 Evergreen 远超，无忧。

**关键问题：PrintToPdf 打印的是 WebView 的“当前顶层文档”，即整个 app（含终端、文件树），不是只有 markdown。** 解决方案：

#### A. 只输出 markdown —— 打印专用 CSS

注入仅对打印生效的样式，隐藏除预览外的一切、并解除预览容器的高度/滚动约束让全文展开：

```css
@media print {
  body > #app > :not(.md-print-isolate) { display: none !important; }
  .md-preview-root { height: auto !important; max-height: none !important; overflow: visible !important; }
  /* 去页眉页脚由 PrintSettings 控制；去分页符： */
  .md-preview-root, .md-preview-root * { break-inside: auto; }
}
```

导出 PDF 前，临时给预览所在容器加 `md-print-isolate` 标记类，导出后移除。

#### B. 拿到 WebView2 指针 —— Wails 打补丁（replace）

Wails v2 不暴露底层 `*edge.Chromium`，`go-webview2 v1.0.22` 也未封装 `PrintToPdf`。链路（已从源码核实可达）：

```
*edge.Chromium → GetController() → GetCoreWebView2() → *ICoreWebView2
  → QueryInterface(IID_ICoreWebView2_7) → PrintToPdf(path, settings, handler)
```

`*edge.Chromium` 在 Wails 的 `internal/frontend/desktop/windows` 里是私有字段。**采用 go.mod 已备好的 `replace` 指令把 wails 指向本地副本**，在 Windows frontend 上加一个导出方法返回该指针（或直接返回 `GetController().GetCoreWebView2()`）。不采用 unsafe 反射（跨版本易碎）。

#### C. 手写 COM（机械但必须）

新增 Go 包 `pdfexport`（Windows-only，`//go:build windows`），仿照 go-webview2 现有 `ComProc`/vtable 模式补：
- `ICoreWebView2_7`（含 `PrintToPdf ComProc`，IID 取自官方 SDK 头文件）
- `ICoreWebView2PrintSettings`（`put_PageWidth/put_PageHeight/put_ScaleFactor/put_ShouldPrintBackgrounds/put_Margin*`）
- `ICoreWebView2PrintToPdfCompletedHandler` 回调（用 channel 把异步转同步）

go-webview2 已用同模式实现十余个接口，移植为确定性机械工作。

#### D. 单页高度测量 + 200 英寸兜底（自动降缩放 + 提示）

前端导出前：在打印布局下测全文高度 `scrollHeight`（px）→ 英寸 `= px / 96`；宽度取固定值（如 A4 宽 8.27in）。

- `heightIn ≤ ~195`：`ScaleFactor=1.0`，`PageHeight=heightIn`。
- `heightIn > ~195`：`scale = clamp(195/heightIn, 0.1, 1.0)`，`PageHeight = heightIn*scale`，并**前端 toast 提示「内容过长，已自动缩放至 N%」**（用户已选此兜底）。
- 极端（scale=0.1 仍超，>1950in）：仍按 0.1 导出并提示可能截断（罕见）。

> 待实现期实测确认：WebView2 在 PageHeight 触顶（>200in）时是报错 `isSuccessful=false` 还是静默截断。

#### E. 流程

```
前端：标记 md-print-isolate → 测高算 PageHeight/ScaleFactor → 调 ExportPdf(savePath, w, h, scale)
后端：SaveFileDialog 取路径 → 设 PrintSettings → ICoreWebView2_7.PrintToPdf → channel 等回调 → 返回成功/失败
前端：移除标记 → toast 结果
```

后端绑定：`func (a *App) ExportPdf(pageWidthIn, pageHeightIn, scaleFactor float64) (string, error)`（路径在内部用 SaveFileDialog 取）。

### 7.3 PDF 先做技术验证（spike）

因 B/C 涉及 Wails 补丁与手写 COM、且 200in 行为官方未明示，**PDF 部分先做一个一次性 spike**，最小化验证三件事，再正式开发：
1. 能从打补丁后的 Wails 取到指针并成功调用 `PrintToPdf` 落一个 PDF；
2. 打印专用 CSS 能让输出**只含 markdown**；
3. 自定义 PageHeight 能出单页，并观察 >200in 的真实表现。

spike 通过后再做正式实现与兜底逻辑。

## 8. 依赖与构建

**前端新增**：`markdown-it`、`markdown-it-anchor`、`markdown-it-checkbox`、`markdown-it-toc-done-right`、`katex`、`mermaid`。（`highlight.js` 已有）。注意 mermaid 体积较大（约数 MB），桌面 app 全本地可接受。

**后端**：`replace` 指向本地 Wails 副本（仅加一个导出方法）；新增 `pdfexport` 包（手写 COM）。`go-webview2` 沿用现有间接版本。

**构建**：`wails build` 流程不变；需确保本地 Wails 副本与 `replace` 路径正确、CI/本机构建可复现。沿用项目既有工具链与 goproxy.cn。

## 9. 错误处理

- 渲染异常：沿用 vscode-office 的降级——整体 try/catch，失败输出转义的 `<pre class="md-render-error">`，不白屏。
- Mermaid 单图失败：就地错误块，不影响其余内容。
- KaTeX 公式错误：KaTeX `throwOnError:false`，渲染成红色错误文本。
- 导出失败：toast 明确报错（路径、权限、PrintToPdf `isSuccessful=false` 等）。
- 文件过大：沿用现有 `ReadFileText` 的 500KB 上限。

## 10. 测试与验证

- **单元测试**（可自动化）：`render.ts` —— 喂典型 markdown（代码块/表格/任务列表/公式/mermaid 占位/TOC），断言输出 HTML 含预期结构与 class。
- **手动验证**（Wails 桌面，按 `/run` 或 `wails dev`）：
  - 打开多种 .md：代码高亮、表格、任务列表勾选、TOC 跳转、外链走系统浏览器、Mermaid 出图、KaTeX 出公式；
  - 18 主题逐一切换，确认只影响预览容器、不污染终端/文件树；重启后主题记忆；
  - HTML 导出：打开导出文件离线可看、含 mermaid SVG；
  - PDF 导出：单页不分页、文字可选可复制、只含 markdown、含主题底色；超长文档触发自动缩放并有提示。

## 11. 分步实施顺序（建议）

1. 装前端依赖；移植 `render.ts` + ext 插件 + `highlight.ts` + `themes.ts`，写单测跑通。
2. 搬运并命名空间化 `themes.css`/`preview.css`；引入 katex css。
3. 新建 `MarkdownPreview.vue`（渲染+主题切换+mermaid 后处理+锚点/外链）；`FilePreview.vue` 的 .md 分支改为委托。跑起来验收渲染与主题。
4. `export-html.ts` + 后端 `SaveExportFile`，打通 HTML 导出。
5. **PDF spike**（§7.3）。
6. PDF 正式实现：Wails replace 补丁 + `pdfexport` COM 包 + 打印 CSS + 测高/兜底 + `ExportPdf` 绑定 + 前端按钮与 toast。
7. 全量手动验证 + 清理。

## 12. 风险与待验证项

| 风险 | 等级 | 缓解 |
|---|---|---|
| 从 Wails 取 `*edge.Chromium` 的最干净路径 | 高 | replace 本地补丁（非 unsafe）；spike 先验证 |
| 手写 ICoreWebView2_7 / PrintSettings COM 的 IID 与 vtable | 中 | 照搬官方 SDK 头文件 GUID + go-webview2 既有模式；spike 验证 |
| PageHeight >200in 的真实行为 | 中 | spike 实测；自动降缩放兜底（已定） |
| 打印 CSS 隔离不彻底（残留 app chrome） | 中 | spike 检查输出；用 `md-print-isolate` 精确标记 |
| themes.css 命名空间化遗漏导致样式泄漏 | 中 | 统一前缀 `.md-preview-root`，对照检查 |
| mermaid 包体积 | 低 | 本地桌面 app 可接受 |
| 维护 Wails 本地副本（replace） | 中 | 仅加一个导出方法，记录补丁点，锁定 wails 版本 |

## 13. 验收标准

- [ ] .md 预览达到 vscode-office 品质：代码高亮、Mermaid、KaTeX、任务列表、TOC、锚点跳转均可用
- [ ] 18 主题可切换、可记忆，且仅作用于预览容器
- [ ] HTML 导出为自包含单文件，离线可看
- [ ] PDF 导出：单页不分页、文字可选可复制、仅含 markdown、保留主题底色
- [ ] 超长文档触发自动缩放并提示
- [ ] cd-sync、终端、文件树等既有功能无回归
