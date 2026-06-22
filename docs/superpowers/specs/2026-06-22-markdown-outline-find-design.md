# Markdown 预览：大纲目录 + Ctrl+F 查找 — 设计文档

- 日期：2026-06-22
- 状态：设计已批准，待编写实现计划
- 参考实现：`D:\E\vscode-office`（`resource/markdown/{outline.js, find.js, preview.css}`）

## 1. 目标

给 SimpleTerminal 的 Markdown 实时预览（`MarkdownPreview.vue`）补齐两项 vscode-office 同款能力：

1. **大纲目录（Outline）**——预览左侧可折叠的标题树面板：滚动联动高亮、推送/浮层两种模式、拖拽调宽、左缘常驻把手。
2. **Ctrl+F 查找（Find）**——预览内查找条：跨节点匹配、命中计数、区分大小写开关、上/下一个 + F3、Esc 关闭，用 CSS Custom Highlight API 着色。

两者**只作用于实时预览**，与导出无关。

## 2. 非目标（Out of Scope）

- 不改动导出三路（HTML / PDF / 长图 PNG），导出产物里不包含大纲面板或查找条。
- 查找不做正则、全词匹配、替换；只提供「区分大小写」开关（与参考一致）。
- 查找状态不持久化（开/关、关键词、当前项均为临时态，刷新即清空，与参考一致）。
- 不引入新的第三方依赖。

## 3. 架构总览（hybrid：复用纯函数 + 原生 Vue UI）

参考实现是无框架依赖的「纯 DOM + CSS」。我们采取**混合**策略：

- **复用其经过验证的纯函数**（无 DOM 依赖、可单测），移植为 TypeScript 工具模块。
- **UI 用原生 Vue 组件**实现，接入既有 `bodyEl`/`scrollEl` 引用与 `localStorage`，吻合本项目「SFC + composable + localStorage」的既有写法，并能借 Vue 响应式正确处理「内容/主题变化后重建」。

## 4. 文件清单

### 新增

| 文件 | 职责 |
|---|---|
| `frontend/src/utils/markdown/outline.ts` | `extractHeadings(bodyEl)`（从 `.md-body` 抠 `h1..h6` 成扁平项）+ 移植纯函数 `buildOutlineTree(items)`（扁平→嵌套树，栈维护祖先链，跳级 h1→h3 仍正确挂载） |
| `frontend/src/utils/markdown/find.ts` | 移植纯函数 `findMatches(full, query, caseSensitive)`（含大小写折叠快/慢路径、`MAX_MATCHES=5000` 上限）、`locateOffset(segs, pos, atEnd)`（全局偏移→文本节点内 (node, offset)，边界归属由 atEnd 决定）+ `buildTextIndex(bodyEl)`（TreeWalker 拼长串与段映射，跳过 `.katex-mathml`） |
| `frontend/src/components/MarkdownOutline.vue` | 大纲面板：递归可折叠树、滚动联动高亮、推送/浮层模式、拖拽调宽、左缘把手 ☰ |
| `frontend/src/components/MarkdownFind.vue` | 查找条：输入框、`N / M` 计数、Aa 区分大小写、↑/↓、✕；CSS Custom Highlight API 着色 |
| `frontend/src/assets/markdown/find-highlight.css` | **全局（非 scoped）** 的 `::highlight(md-find)` / `::highlight(md-find-current)` 规则——scoped CSS 无法命中 `::highlight` 伪元素，必须全局引入 |

### 修改

| 文件 | 改动 |
|---|---|
| `frontend/src/components/MarkdownPreview.vue` | 暴露 `scrollEl`；在 `.md-preview-root` 内挂载 `MarkdownOutline` 与 `MarkdownFind`；工具条新增 📑（大纲）与 🔍（查找）按钮；持有并持久化大纲 `open`/`mode`/`width` 状态；挂载 Ctrl+F 快捷键监听 |

### 测试

| 文件 | 覆盖 |
|---|---|
| `frontend/src/__tests__/outline.test.ts` | `buildOutlineTree`：常规嵌套、跳级（h1→h3）、空输入 |
| `frontend/src/__tests__/find.test.ts` | `findMatches`：区分/不区分大小写、Unicode 折叠慢路径、`MAX_MATCHES` 截断；`locateOffset`：起点/终点边界归属 |

## 5. 数据流

### 5.1 大纲

1. `MarkdownPreview` 暴露 `bodyEl`（`.md-body`）与 `scrollEl`（`.md-scroll`，真正的滚动容器）。
2. `MarkdownOutline` 接收 `bodyEl`、`scrollEl`、`html`（变化时触发重建）、以及 `open`/`mode`/`width`（父持有、可双向更新）。
3. `html` 变化后（`nextTick`）：扫描 `bodyEl` 内 `h1..h6`，保留「文本非空且有 `id`」者 → 扁平项 → `buildOutlineTree` → 递归渲染嵌套节点（caret 折叠）。
4. 点击标题链接 → `document.getElementById(id).scrollIntoView({ behavior:'smooth', block:'start' })`（自动滚动 `.md-scroll`）。
5. 滚动联动高亮：`IntersectionObserver({ root: scrollEl, rootMargin:'0px 0px -70% 0px', threshold:0 })` 观察各标题元素；并对 **`scrollEl`**（非 window）监听 `scroll` → `pickActive()`：设置当前 `.active` 链接、自动展开其被折叠的祖先、把活跃链接滚进列表可视区。
6. 拖拽调宽：右缘 grip 拖动 → 更新 CSS 变量 `--md-outline-w`（夹在 `180 .. min(480, 面板可用宽)`），`mouseup` 时回写 `localStorage`。
7. 无标题文档：父侧工具栏 📑 置灰禁用 + 点按提示「无标题」；面板与左缘把手不渲染、不推送正文。

### 5.2 查找

1. `MarkdownFind` 接收 `bodyEl`、`scrollEl`、`open`（父持有）；关闭时不渲染高亮。
2. 打开入口：① Ctrl/⌘+F（capture 阶段，挂在预览根，见 §7.5），用当前选区预填、聚焦并全选输入框；② 工具栏 🔍 按钮。
3. 输入（防抖 120ms）：`buildTextIndex(bodyEl)`（TreeWalker 遍历文本节点，跳过 `.katex-mathml`）→ `findMatches` → 对每个命中用 `locateOffset` 求起止 (node, offset) → 建 `Range` → 加入 `CSS.highlights` 的 `md-find`（全部）与 `md-find-current`（当前项）。计数显示 `N / M`，截断时显示 `M+`。
4. Enter/↓/F3 下一个，Shift+Enter/↑/Shift+F3 上一个（环绕）；当前项经 `.md-scroll` 滚动居中（命中元素 `scrollIntoView({ block:'center' })`，已在舒适区则不滚）。
5. Esc / ✕ 关闭：清空高亮、归还焦点给文档（关闭后键盘仍可滚动正文）。
6. Aa 区分大小写开关：切换后重跑搜索。
7. 晚到内容（图片/KaTeX/Mermaid 异步渲染）：监听 `window` `load`，若查找条开着且有词则重跑一次（不滚动，避免跳动）。

## 6. 状态与持久化

`localStorage` 键，沿用既有 `md-preview-` 前缀：

| 键 | 值 | 默认 |
|---|---|---|
| `md-preview-outline-open` | `'0'` / `'1'` | `'0'`（首次默认关闭，不遮挡内容；之后记忆） |
| `md-preview-outline-mode` | `'push'` / `'overlay'` | `'push'` |
| `md-preview-outline-width` | px 数字字符串 | `260` |

查找状态全程不持久化。

## 7. 定位与样式适配（SimpleTerminal 特有，关键）

参考实现里整个 webview 就是预览，元素挂 `document.body` 且 `position:fixed`、滚动用 `window`。本项目预览只是「文件树 + 预览 + 终端」中的一个可缩放子面板，必须改造：

### 7.1 fixed → absolute，且挂在 `.md-preview-root` 内
大纲面板、左缘把手、查找条全部 `position:absolute`，挂在 `.md-preview-root`（本就 `position:relative; overflow:hidden`）内部。好处：① 不溢出预览面板去盖住终端；② 自动继承 `--md-ui-bg / --md-ui-border / --md-fg / --md-muted / --md-pre-bg / --md-link` 等主题变量（这些变量定义在 `.md-preview-root[data-theme=...]`）。`overflow:hidden` 也使关闭态 `translateX(-100%)` 的面板被裁掉、不外露。

### 7.2 滚动联动监听 `.md-scroll` 而非 `window`
`.md-scroll` 是 `overflow:auto` 的滚动容器，其 `scroll` 事件**不冒泡到 window**。故 `scroll` 监听与 `IntersectionObserver` 的 `root` 都用 `scrollEl`。`scrollIntoView` 本身会就近滚动 `.md-scroll`，无需特殊处理（既有锚点点击已验证可行）。

### 7.3 推送模式作用于 `.md-scroll`
推送模式打开时，给 `.md-scroll` 加 `padding-left: var(--md-outline-w)`（带 transition），而非参考里的 `body padding-left`。浮层模式则面板盖在正文上、不挤压。CSS 变量 `--md-outline-w` 由 `MarkdownPreview` 设在 `rootEl.style` 上（启动时从 localStorage 初始化）。

### 7.4 工具栏与把手
- 工具条（现 `.md-toolbar`，右下角）新增两枚按钮：📑 切换大纲、🔍 打开查找；插在既有 🎨/📤/➖/➕/🔄 之列。
- 左缘把手 ☰（`absolute; left:0; top:50%`）仅在大纲关闭时显示，点按展开——对齐 vscode-office。

### 7.5 Ctrl+F 与终端隔离
快捷键监听挂在**预览根元素**（capture 阶段），并给 `.md-scroll` 加 `tabindex`（使预览可聚焦）。如此「仅当焦点在预览内」Ctrl+F 才触发查找——焦点在终端（xterm）时事件不会到达预览根，天然避免与终端按键处理冲突。🔍 工具栏按钮不依赖焦点，可直接打开。

### 7.6 缩放无坐标隐患
本项目正文缩放用 `font-size`（非 CSS `zoom`），故参考实现中为规避 `zoom` 坐标串台而保留的复杂处理在此不需要——查找的 `getBoundingClientRect` 与滚动数学直接可用。

### 7.7 着色与降级
目标平台是 Win11 常青 WebView2，**CSS Custom Highlight API 必然可用**。据此**移除参考里的旧版 `#md-find-overlay` 浮层降级路径**（YAGNI），仅保留 `::highlight` 着色，显著简化 `find.ts` 与查找组件。`::highlight(md-find)` / `::highlight(md-find-current)` 规则放入全局 `find-highlight.css`（scoped 无法命中 `::highlight`），颜色用 `var(--md-find-bg, rgba(255,210,0,0.55))` / `var(--md-find-current-bg, rgba(255,145,0,0.9))` 并强制深色文字保证亮/暗主题对比度。

## 8. 组件接口（供实现计划参考）

```
MarkdownPreview.vue（父，新增状态）
  ref outlineOpen / outlineMode / outlineWidth  ←→ localStorage
  ref findOpen
  工具栏 📑 → outlineOpen 取反（无标题时禁用）
  工具栏 🔍 → findOpen = true
  rootEl 上 capture 监听 Ctrl/⌘+F → findOpen = true（预填选区）
  设 rootEl.style['--md-outline-w'] = outlineWidth

MarkdownOutline.vue
  props : bodyEl, scrollEl, html, open, mode, width
  emits : update:open, update:mode, update:width
  内部  : 标题扫描+建树、递归节点渲染、IntersectionObserver(root=scrollEl)+scrollEl scroll 联动、
          caret 折叠、左缘把手 ☰、头部（标题/⇆ 模式/✕ 关闭）、右缘 grip 调宽

MarkdownFind.vue
  props : bodyEl, scrollEl, open
  emits : update:open
  暴露  : 打开时聚焦输入框（watch open）
  内部  : 查找条 UI、防抖搜索、Range + CSS.highlights、计数、↑/↓/F3/Esc、Aa
```

## 9. 边界情况

- **无标题**：见 §5.1.7（📑 禁用 + 提示，面板不渲染）。
- **标题缺 `id`**：`markdown-it-anchor` 默认给全部标题加 id，仍做防御（大纲跳过该项；查找仍按其文本工作）。
- **内容/主题重渲**：大纲随 `html` 变化重建；查找在下次键入时重建索引，并在 `load` 时把晚到内容纳入（不滚动）。
- **面板被拖到比大纲还窄**：宽度上限夹 `min(480, 面板可用宽)`；推送模式下正文照常横向滚动。
- **Mermaid 异步渲染**：查找经 `window load` 重跑覆盖；首屏「开框→输入」恰落在 Mermaid 完成前的极短窗口会先索引到源码，下一次按键即自愈（与参考一致）。
- **预览上下/左右布局切换**：面板 `absolute` 于 `.md-preview-root`，两种布局均正确锚定。

## 10. 测试计划

- **单元（vitest）**：`buildOutlineTree`（嵌套、跳级、空）；`findMatches`（大小写两路 + Unicode 折叠慢路径 + 5000 截断）；`locateOffset`（起点/终点边界归属）。`npm run build` 做类型校验。
- **手动**：跨多套主题、两种预览布局（右/上）、窄面板、无标题文档，逐项验证大纲（导航/联动/调宽/模式切换/把手）与查找（计数/导航/区分大小写/Esc/选区预填）。

## 11. 已知小限

- 预览在外部改动/刷新时整体重渲，查找条状态（词、当前项）随之重置——只读预览可接受，不持久化以避免跨重建的额外状态管线（与参考一致）。
