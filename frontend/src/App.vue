<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount } from 'vue'
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

// ── 自适应视口 ───────────────────────────────────────────
// 测量 .split 容器，渲染时把预览尺寸钳进可用空间，保证另一侧（终端）至少保留
// TERMINAL_MIN，避免小窗口下某一侧被挤成 0；窗口放大时预览可回到用户期望尺寸。
const TERMINAL_MIN = 120
const DIVIDER_SIZE = 4

const splitRef = ref<HTMLElement | null>(null)
const splitW = ref(0)
const splitH = ref(0)
let splitRO: ResizeObserver | null = null

onMounted(() => {
  const el = splitRef.value
  if (!el) return
  splitW.value = el.clientWidth
  splitH.value = el.clientHeight
  splitRO = new ResizeObserver(() => {
    splitW.value = el.clientWidth
    splitH.value = el.clientHeight
  })
  splitRO.observe(el)
})
onBeforeUnmount(() => {
  splitRO?.disconnect()
  splitRO = null
})

// 预览在某方向上的最大可用尺寸（容器减终端最小值与分隔条）；容器未测量时不设上限。
function previewMax(container: number, min: number) {
  return container > 0 ? Math.max(min, container - TERMINAL_MIN - DIVIDER_SIZE) : Infinity
}
// 渲染用的有效尺寸：用户期望值钳进当前视口；窗口放大时可回到期望值。
const effPreviewWidth = computed(() =>
  Math.min(Math.max(PREVIEW_MIN_W, previewWidth.value), previewMax(splitW.value, PREVIEW_MIN_W)))
const effPreviewHeight = computed(() =>
  Math.min(Math.max(PREVIEW_MIN_H, previewHeight.value), previewMax(splitH.value, PREVIEW_MIN_H)))

// 预览栏与终端共用同一条分隔条：左右布局拖宽度、上下布局拖高度。
// 拖动上限钳到当前视口（终端至少留 TERMINAL_MIN），分隔条拖到边缘即停。
function onPreviewResize(delta: number) {
  if (previewLayout.value === 'horizontal') {
    // 分隔条在预览栏左侧，向左拖（delta 负）→ 预览变宽
    const max = previewMax(splitW.value, PREVIEW_MIN_W)
    previewWidth.value = Math.min(max, Math.max(PREVIEW_MIN_W, previewWidth.value - delta))
    localStorage.setItem(PREVIEW_W_KEY, String(previewWidth.value))
  } else {
    // 分隔条在预览栏下方，向下拖（delta 正）→ 预览变高
    const max = previewMax(splitH.value, PREVIEW_MIN_H)
    previewHeight.value = Math.min(max, Math.max(PREVIEW_MIN_H, previewHeight.value + delta))
    localStorage.setItem(PREVIEW_H_KEY, String(previewHeight.value))
  }
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

// 终端目录变化时，左侧文件树跟随切到该目录。
// path 来自 PowerShell 提示符，已是解析好的真实绝对路径，直接 loadDir。
function onTerminalCwd(path: string) {
  if (path === currentPath.value) return // 已是当前根则跳过（兼防选目录→发cd→回灌的循环）
  void loadDir(path)
}
</script>

<template>
  <div class="layout">
    <div class="tree-pane" :style="{ width: treeWidth + 'px' }">
      <FileTree />
    </div>
    <Divider @resize="onResize" />
    <div class="main-area">
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
      <!--
        终端 + 预览栏共用一个容器：左右布局 → flex row，上下布局 → flex column。
        预览栏始终是同一个 FilePreview 实例（上下布局仅靠 CSS order 反转到顶部），
        切换布局只改 CSS、不卸载组件，因此不会重新读取/高亮文件，切换即时完成。
      -->
      <div ref="splitRef" class="split" :class="previewLayout">
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
        <template v-if="selectedFile">
          <Divider
            class="preview-divider"
            :orientation="previewLayout === 'horizontal' ? 'vertical' : 'horizontal'"
            @resize="onPreviewResize"
          />
          <FilePreview
            class="preview-pane"
            :file="selectedFile"
            :placement="previewLayout === 'horizontal' ? 'right' : 'top'"
            :style="previewLayout === 'horizontal'
              ? { width: effPreviewWidth + 'px' }
              : { height: effPreviewHeight + 'px' }"
            @close="selectedFile = null"
          />
        </template>
      </div>
    </div>
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
.main-area {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
}
/* 终端 + 预览栏的可切换容器：左右=row，上下=column */
.split {
  flex: 1 1 0;
  min-width: 0;
  min-height: 0;
  display: flex;
}
.split.horizontal { flex-direction: row; }
.split.vertical { flex-direction: column; }
/* 上下布局：预览在上、终端在下——保持 DOM 顺序不变，仅用 order 反转 */
.split.vertical .preview-pane { order: 0; }
.split.vertical .preview-divider { order: 1; }
.split.vertical .terminal-panels { order: 2; }

.terminal-panels {
  flex: 1 1 0;
  min-width: 0;
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
