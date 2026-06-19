# 更新日志 / Changelog

本项目所有重要变更都会记录在此文件。
All notable changes to this project are documented in this file.

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

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
