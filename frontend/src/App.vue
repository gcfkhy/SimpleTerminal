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

const MIN_WIDTH = 160
const MAX_WIDTH = 500
const WIDTH_KEY = 'treeWidth'

function clamp(w: number): number {
  return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, w))
}

const saved = Number(localStorage.getItem(WIDTH_KEY))
const treeWidth = ref<number>(Number.isFinite(saved) && saved > 0 ? clamp(saved) : 280)

function onResize(deltaX: number) {
  treeWidth.value = clamp(treeWidth.value + deltaX)
  localStorage.setItem(WIDTH_KEY, String(treeWidth.value))
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
    <FilePreview
      v-if="selectedFile"
      :file="selectedFile"
      @close="selectedFile = null"
    />
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
