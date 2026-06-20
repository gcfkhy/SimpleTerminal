# 更新日志 / Changelog

本项目所有重要变更都会记录在此文件。
All notable changes to this project are documented in this file.

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [v1.3.0] · 2026-06-20

### ✨ 新增 (Added)

- **Markdown 预览全面重做** — 渲染引擎从 `marked` 换成 `markdown-it`，对齐 vscode-office 的阅读体验：
  - **代码语法高亮**（修复此前 md 内代码块不高亮）、表格、任务列表、自动目录 (TOC)、标题锚点跳转；
  - **Mermaid 流程图**与 **KaTeX 数学公式**；
  - **18 套亮 / 暗主题**一键切换（Catppuccin、GitHub、Dracula、Nord、Tokyo Night、Solarized、Gruvbox…），全局记忆，且仅作用于预览容器，不影响终端与文件树；
  - **导出菜单**（预览右下角）：**PDF（单页不分页、文字可选可复制）/ HTML（自包含单文件）/ 长图 PNG**，外观还原当前主题；
  - **缩放**（➖ / ➕ 按钮与 Ctrl+滚轮，居中显示比例、点击还原 100%，记忆）与**刷新**（重读当前文件重渲）。

---

### ✨ Added

- **Markdown preview fully reworked** — The rendering engine moves from `marked` to `markdown-it`, matching the vscode-office reading experience:
  - **Code syntax highlighting** (fixing code blocks that previously rendered without highlighting), tables, task lists, an auto table of contents (TOC), and heading anchor navigation;
  - **Mermaid diagrams** and **KaTeX math formulas**;
  - **18 light / dark themes** with one-click switching (Catppuccin, GitHub, Dracula, Nord, Tokyo Night, Solarized, Gruvbox…), remembered globally and scoped to the preview only, leaving the terminal and file tree untouched;
  - **Export menu** (bottom-right of the preview): **PDF (single continuous page, selectable & copyable text) / HTML (self-contained single file) / long-image PNG**, reproducing the current theme;
  - **Zoom** (➖ / ➕ buttons and Ctrl+wheel, with a centered percentage indicator, click to reset to 100%, remembered) and **Refresh** (re-read and re-render the current file).

---

## [v1.2.2] · 2026-06-19

### 🐛 修复 (Fixed)

- **Ctrl+V 粘贴在 claude 等 TUI 程序中失效** — xterm 默认把 Ctrl+V 当控制字符 `\x16` 发给程序，PowerShell 靠 PSReadLine 自行把 `\x16` 绑成粘贴才碰巧能用，而 claude 等 TUI 不认 `\x16`，表现为「按 Ctrl+V 没反应」。现在拦截 Ctrl+V，改走浏览器原生粘贴（与右键菜单同一条路径，正确携带 bracketed paste），在 PowerShell / claude 等各类程序中都能粘贴。
- **Ctrl+C 复制终端选中内容** — 选中文本时 Ctrl+C 将选区复制到剪贴板（经 Wails Go 侧剪贴板，打包后的 `wails://` 非安全上下文也可靠）；未选中文本时仍照常发送中断信号（SIGINT），不影响打断正在运行的程序。行为与 Windows Terminal 一致。

---

### 🐛 Fixed

- **Ctrl+V paste did nothing in TUI apps like claude** — xterm sends Ctrl+V to the program as the control char `\x16`; PowerShell only worked because PSReadLine binds `\x16` to paste, while TUIs such as claude ignore it ("Ctrl+V does nothing"). Ctrl+V is now intercepted and routed through the browser's native paste — the same path as the right-click menu, correctly carrying bracketed paste — so pasting works across PowerShell, claude, and other programs.
- **Ctrl+C copies the terminal selection** — With text selected, Ctrl+C copies the selection to the clipboard (via the Wails Go-side clipboard, reliable even in the packaged `wails://` non-secure context); with nothing selected it still sends the interrupt signal (SIGINT) as before, so interrupting a running program is unaffected. Matches Windows Terminal behavior.

---

## [v1.2.1] · 2026-06-19

### 🐛 修复 (Fixed)

- **缩放窗口卡顿** — 终端的 `ResizeObserver` 不再每次回调都同步 `fit()` + ConPTY resize：用 `requestAnimationFrame` 合并每帧多次回调，且仅在终端行列数真正变化时、经 80ms 防抖后再发起 PTY resize，消除缩放时的渲染 / IPC 洪泛。
- **切换布局卡顿** — 终端与预览栏改为共用同一容器、预览栏始终是同一个组件实例（仅靠 CSS 反转方向），切换左右 / 上下布局不再卸载重建预览栏，因而不再重新读取与高亮文件，切换瞬间完成。
- **小窗口下布局不自适应** — 预览栏渲染尺寸现在会按当前视口钳制，始终为终端保留至少 120px，避免非最大化窗口下某一侧被挤成 0；窗口放大时预览恢复到用户设定尺寸。

---

### 🐛 Fixed

- **Laggy window resizing** — The terminal's `ResizeObserver` no longer runs `fit()` + a ConPTY resize on every callback: callbacks within a frame are coalesced via `requestAnimationFrame`, and the PTY resize is sent only when the terminal's columns/rows actually change, after an 80 ms debounce — eliminating the render / IPC flood during resize.
- **Laggy layout switching** — The terminal and preview pane now share one container and the preview is always the same component instance (only its direction is flipped via CSS), so toggling between side-by-side and stacked layouts no longer unmounts/recreates the preview — the file is not re-read or re-highlighted, and switching is instant.
- **Layout not adapting in small windows** — The preview pane's rendered size is now clamped to the current viewport, always reserving at least 120 px for the terminal, so neither pane collapses to zero in a non-maximized window; the preview returns to your chosen size when the window grows.

---

## [v1.2.0] · 2026-06-19

### ✨ 新增 (Added)

- **预览栏左右 / 上下布局切换** — 导航栏（标签栏）右侧新增两个布局图标，一键在两种布局间切换，当前布局蓝色高亮：
  - **左右布局**：预览栏停靠在窗口右侧，与终端左右并排；
  - **上下布局**：预览栏在上、终端在下，左侧文件树保持整列满高。
- **拖拽无尺寸上限** — 移除预览栏原先的 700px 宽度上限；左右布局的宽度、上下布局的高度均可拖到任意大小（仅保留下限防止塌缩）。两种布局的尺寸各自独立记忆并持久化到 `localStorage`。

---

### ✨ Added

- **Toggle the preview pane between side-by-side and stacked layouts** — Two layout icons on the right of the nav (tab) bar switch the preview pane with one click; the active layout is highlighted in blue:
  - **Side-by-side**: the preview pane docks to the right of the terminal;
  - **Stacked**: the preview sits on top with the terminal below, while the left file tree keeps its full height.
- **No maximum drag size** — Removed the former 700px width cap on the preview pane; the width (side-by-side) and height (stacked) can be dragged to any size, keeping only a minimum to prevent collapse. Each layout remembers its own size, persisted to `localStorage`.

---

## [v1.1.0] · 2026-06-19

![SimpleTerminal v1.1.0 截图 / screenshot](assets/screenshot.png)

> 截图展示：终端 `cd` 后左侧文件树自动同步、标签栏的两个新建按钮（`+` 默认目录 / 📁 当前目录）。
> The screenshot shows the file tree auto-syncing after a terminal `cd`, and the two new-tab buttons (`+` default dir / 📁 current dir).

### ✨ 新增 (Added)

- **终端 `cd` 自动同步文件树** — 应用启动后、左侧尚未选择目录时，在终端里首次切换目录（`cd`）会自动把左侧文件树加载到该目录。当前路径通过解析 PowerShell 提示符识别，因此对**手动输入、粘贴、↑ 历史、`Tab` 补全**等各种输入方式都生效。一旦文件树有了内容，此自动同步即关闭，之后的 `cd` 不再改动文件树。
- **双击重命名标签页** — 双击任意标签页即可就地重命名：输入框自动全选，`Enter` 或失焦保存，`Esc` 取消；留空则保持原名。
- **两个「新建标签页」按钮** — 标签栏左上角拆分为两个按钮，悬停均有说明文字：
  - `+`：在**默认目录**（用户主目录）新建标签页；
  - 📁 文件夹图标：在**当前目录**（左侧文件树的当前路径）新建标签页；文件树为空时退化为默认目录。

### 🛠 改进 (Changed)

- 终端启动目录显式固定为用户主目录（基于 ConPTY `WorkDir`），与 Windows 默认行为一致，并为首次 `cd` 的路径解析提供确定基准。

---

### ✨ Added

- **Terminal `cd` auto-syncs the file tree** — After launch, while no directory has been picked on the left, the first `cd` in the terminal automatically loads the file tree to that directory. The current path is detected by parsing the PowerShell prompt, so it works regardless of how the path was entered — **typing, pasting, ↑ history, or `Tab` completion**. Once the tree has content, the auto-sync turns off and later `cd`s no longer move the tree.
- **Double-click to rename a tab** — Double-click any tab to rename it in place: the text is auto-selected, `Enter` or blur saves, `Esc` cancels, and an empty value keeps the original name.
- **Two "new tab" buttons** — The top-left of the tab bar is split into two buttons, each with a hover tooltip:
  - `+` — opens a new tab in the **default directory** (the user home folder);
  - 📁 folder icon — opens a new tab in the **current directory** (the file tree's current path); falls back to the default directory when the tree is empty.

### 🛠 Changed

- The terminal now starts explicitly in the user home directory (via ConPTY `WorkDir`), matching Windows' default behavior and giving the first `cd` a deterministic base for path resolution.

---

## [V1.0] · 2026

首个公开版本：文件目录树、真实 PowerShell 终端、拖拽路径注入、多标签页、文件预览侧栏（图片 / 代码 / Markdown / 视频）、可拖拽面板、Catppuccin Mocha 主题。

First public release: file directory tree, real PowerShell terminal, drag-to-terminal path injection, multi-tab sessions, file preview sidebar (images / code / Markdown / video), resizable panels, Catppuccin Mocha theme.
