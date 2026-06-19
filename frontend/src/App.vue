<script setup lang="ts">
import { ref, watch } from 'vue'
import FileTree from './components/FileTree.vue'
import FilePreview from './components/FilePreview.vue'
import Divider from './components/Divider.vue'
import Terminal from './components/Terminal.vue'
import TabBar from './components/TabBar.vue'
import { useFileTree } from './composables/useFileTree'
import { EventsEmit } from '../wailsjs/runtime'
import { ClosePty } from '../wailsjs/go/main/App'

// ── 左侧文件树宽度 ─────────────────────────────────────────
const TREE_MIN = 160
const TREE_MAX = 500
const TREE_KEY = 'treeWidth'

const savedTree = Number(localStorage.getItem(TREE_KEY))
const treeWidth = ref<number>(Number.isFinite(savedTree) && savedTree > 0
  ? Math.min(TREE_MAX, Math.max(TREE_MIN, savedTree)) : 280)

function onResize(delta: number) {
  treeWidth.value = Math.min(TREE_MAX, Math.max(TREE_MIN, treeWidth.value + delta))
  localStorage.setItem(TREE_KEY, String(treeWidth.value))
}

// ── 预览栏布局方向（左右 / 上下）─────────────────────────
type PreviewLayout = 'horizontal' | 'vertical'
const LAYOUT_KEY = 'previewLayout'
const savedLayout = localStorage.getItem(LAYOUT_KEY)
const previewLayout = ref<PreviewLayout>(savedLayout === 'vertical' ? 'vertical' : 'horizontal')

function setPreviewLayout(layout: PreviewLayout) {
  previewLayout.value = layout
  localStorage.setItem(LAYOUT_KEY, layout)
}

// ── 预览栏宽度（左右布局）/ 高度（上下布局）─────────────
// 仅保留下限防止塌缩，上限不限制（可拖到任意大）。
const PREVIEW_MIN_W = 200
const PREVIEW_MIN_H = 120
const PREVIEW_W_KEY = 'previewWidth'
const PREVIEW_H_KEY = 'previewHeight'

const savedPreviewW = Number(localStorage.getItem(PREVIEW_W_KEY))
const previewWidth = ref<number>(Number.isFinite(savedPreviewW) && savedPreviewW > 0
  ? Math.max(PREVIEW_MIN_W, savedPreviewW) : 320)

const savedPreviewH = Number(localStorage.getItem(PREVIEW_H_KEY))
const previewHeight = ref<number>(Number.isFinite(savedPreviewH) && savedPreviewH > 0
  ? Math.max(PREVIEW_MIN_H, savedPreviewH) : 260)

function onPreviewResize(delta: number) {
  // 分隔线在预览栏左侧，向左拖（delta 负）→ 预览变宽
  previewWidth.value = Math.max(PREVIEW_MIN_W, previewWidth.value - delta)
  localStorage.setItem(PREVIEW_W_KEY, String(previewWidth.value))
}

function onPreviewHeightResize(delta: number) {
  // 分隔线在预览栏下方，向下拖（delta 正）→ 预览变高
  previewHeight.value = Math.max(PREVIEW_MIN_H, previewHeight.value + delta)
  localStorage.setItem(PREVIEW_H_KEY, String(previewHeight.value))
}

// ── Tab 管理 ──────────────────────────────────────────────
interface Tab {
  id: string
  title: string
  initialDir?: string
}

let counter = 1
const tabs = ref<Tab[]>([{ id: 'tab-1', title: 'Terminal 1' }])
const activeTabId = ref('tab-1')

function addTab(initialDir?: string) {
  counter++
  const id = `tab-${counter}`
  tabs.value.push({
    id,
    title: `Terminal ${counter}`,
    initialDir,
  })
  activeTabId.value = id
}

// 在「当前目录」（左侧文件树路径）新建标签页；树为空时退化为默认目录。
function addTabHere() {
  addTab(currentPath.value || undefined)
}

function closeTab(id: string) {
  if (tabs.value.length <= 1) return
  const idx = tabs.value.findIndex((t) => t.id === id)
  tabs.value.splice(idx, 1)
  void ClosePty(id)
  if (activeTabId.value === id) {
    activeTabId.value = tabs.value[Math.max(0, idx - 1)].id
  }
}

function activateTab(id: string) {
  activeTabId.value = id
}

function renameTab(id: string, title: string) {
  const tab = tabs.value.find((t) => t.id === id)
  if (tab) tab.title = title
}

// 选择目录后向当前活跃 Tab 发送 cd
const { pickedDir, selectedFile, currentPath, loadDir } = useFileTree()
watch(pickedDir, (dir) => {
  if (dir) {
    EventsEmit('pty:input', activeTabId.value, `cd "${dir}"\r`)
  }
})

// 终端目录变化同步左侧树：仅当左侧树为空（未选过目录）时生效。
// path 来自 PowerShell 提示符，已是解析好的真实绝对路径，直接 loadDir。
// 命中后 loadDir 会填上 currentPath，gate 自动关闭，后续目录变化不再驱动树。
function onTerminalCwd(path: string) {
  if (currentPath.value !== '') return // 树已有内容，gate 关闭
  void loadDir(path)
}
</script>

<template>
  <div class="layout">
    <div class="tree-pane" :style="{ width: treeWidth + 'px' }">
      <FileTree />
    </div>
    <Divider @resize="onResize" />
    <div class="terminal-area">
      <TabBar
        :tabs="tabs"
        :activeId="activeTabId"
        :previewLayout="previewLayout"
        @add="addTab"
        @add-here="addTabHere"
        @close="closeTab"
        @activate="activateTab"
        @rename="renameTab"
        @set-layout="setPreviewLayout"
      />
      <!-- 上下布局：预览栏在上方，终端在下面（左侧文件树保持满高）-->
      <template v-if="selectedFile && previewLayout === 'vertical'">
        <FilePreview
          :file="selectedFile"
          placement="top"
          :style="{ height: previewHeight + 'px' }"
          @close="selectedFile = null"
        />
        <Divider orientation="horizontal" @resize="onPreviewHeightResize" />
      </template>
      <div class="terminal-panels">
        <div
          v-for="tab in tabs"
          :key="tab.id"
          class="terminal-slot"
          :class="{ active: tab.id === activeTabId }"
        >
          <Terminal
            :tabId="tab.id"
            :isActive="tab.id === activeTabId"
            :initialDir="tab.initialDir"
            @cwd="onTerminalCwd"
          />
        </div>
      </div>
    </div>
    <!-- 左右布局：预览栏在最右侧 -->
    <template v-if="selectedFile && previewLayout === 'horizontal'">
      <Divider @resize="onPreviewResize" />
      <FilePreview
        :file="selectedFile"
        placement="right"
        :style="{ width: previewWidth + 'px' }"
        @close="selectedFile = null"
      />
    </template>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
.tree-pane {
  flex: 0 0 auto;
  height: 100%;
  overflow: hidden;
}
.terminal-area {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
}
.terminal-panels {
  flex: 1 1 0;
  min-height: 0;
  position: relative;
}
.terminal-slot {
  position: absolute;
  inset: 0;
  display: none;
}
.terminal-slot.active {
  display: flex;
}
</style>
