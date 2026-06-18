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

function onResize(deltaX: number) {
  treeWidth.value = Math.min(TREE_MAX, Math.max(TREE_MIN, treeWidth.value + deltaX))
  localStorage.setItem(TREE_KEY, String(treeWidth.value))
}

// ── 右侧预览栏宽度 ─────────────────────────────────────────
const PREVIEW_MIN = 200
const PREVIEW_MAX = 700
const PREVIEW_KEY = 'previewWidth'

const savedPreview = Number(localStorage.getItem(PREVIEW_KEY))
const previewWidth = ref<number>(Number.isFinite(savedPreview) && savedPreview > 0
  ? Math.min(PREVIEW_MAX, Math.max(PREVIEW_MIN, savedPreview)) : 320)

function onPreviewResize(deltaX: number) {
  // 分隔线在预览栏左侧，向左拖（deltaX 负）→ 预览变宽
  previewWidth.value = Math.min(PREVIEW_MAX, Math.max(PREVIEW_MIN, previewWidth.value - deltaX))
  localStorage.setItem(PREVIEW_KEY, String(previewWidth.value))
}

// ── Tab 管理 ──────────────────────────────────────────────
interface Tab {
  id: string
  title: string
  initialDir?: string
}

let counter = 1
const initialDir = localStorage.getItem('lastDir') ?? undefined
const tabs = ref<Tab[]>([{ id: 'tab-1', title: 'Terminal 1', initialDir }])
const activeTabId = ref('tab-1')

function addTab() {
  counter++
  const id = `tab-${counter}`
  tabs.value.push({
    id,
    title: `Terminal ${counter}`,
    initialDir: localStorage.getItem('lastDir') ?? undefined,
  })
  activeTabId.value = id
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

// 选择目录后向当前活跃 Tab 发送 cd
const { pickedDir, selectedFile } = useFileTree()
watch(pickedDir, (dir) => {
  if (dir) {
    EventsEmit('pty:input', activeTabId.value, `cd "${dir}"\r`)
  }
})
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
        @add="addTab"
        @close="closeTab"
        @activate="activateTab"
      />
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
          />
        </div>
      </div>
    </div>
    <template v-if="selectedFile">
      <Divider @resize="onPreviewResize" />
      <FilePreview
        :file="selectedFile"
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
