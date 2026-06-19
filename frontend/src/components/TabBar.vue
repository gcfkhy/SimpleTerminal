<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

interface Tab {
  id: string
  title: string
}
const props = defineProps<{ tabs: Tab[]; activeId: string }>()
const emit = defineEmits<{
  add: []
  addHere: []
  close: [id: string]
  activate: [id: string]
  rename: [id: string, title: string]
}>()

const scrollRef = ref<HTMLElement | null>(null)

// ── 双击重命名 ────────────────────────────────────────────
// 同一时刻只有一个选项卡处于编辑态。
const editingId = ref<string | null>(null)
const editText = ref('')

function startEdit(tab: Tab) {
  editingId.value = tab.id
  editText.value = tab.title
  void nextTick(() => {
    // 此刻页面上只有一个 .tab-edit，直接查询并聚焦全选
    const el = scrollRef.value?.querySelector<HTMLInputElement>('.tab-edit')
    el?.focus()
    el?.select()
  })
}

function commitEdit() {
  if (editingId.value === null) return
  const id = editingId.value
  editingId.value = null // 先退出编辑态，避免随后的 blur 重复提交
  const name = editText.value.trim()
  const tab = props.tabs.find((t) => t.id === id)
  if (name && tab && name !== tab.title) {
    emit('rename', id, name)
  }
  // name 为空或未变：静默还原（不改标题）
}

function cancelEdit() {
  editingId.value = null
}

watch(() => props.activeId, async () => {
  await nextTick()
  const el = scrollRef.value?.querySelector<HTMLElement>('.tab.active')
  el?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
})

function onWheel(e: WheelEvent) {
  e.preventDefault()
  const idx = props.tabs.findIndex((t) => t.id === props.activeId)
  const delta = e.deltaY > 0 ? 1 : -1
  const next = props.tabs[idx + delta]
  if (next) emit('activate', next.id)
}
</script>

<template>
  <div class="tabbar">
    <button
      class="tab-add"
      title="新建标签页 · 默认目录"
      @click="emit('add')"
    >+</button>
    <button
      class="tab-add tab-add-here"
      title="新建标签页 · 当前目录（左侧文件树）"
      @click="emit('addHere')"
    >
      <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
        <path
          d="M3 6.5A1.5 1.5 0 0 1 4.5 5h4l1.6 1.8h7.4A1.5 1.5 0 0 1 19 8.3V17a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 3 17V6.5Z"
          fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"
        />
        <path d="M14.5 13h4M16.5 11v4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
      </svg>
    </button>
    <div ref="scrollRef" class="tabs-scroll" @wheel="onWheel">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: tab.id === activeId }"
        @click="emit('activate', tab.id)"
        @dblclick="startEdit(tab)"
      >
        <input
          v-if="tab.id === editingId"
          class="tab-edit"
          v-model="editText"
          @click.stop
          @dblclick.stop
          @keydown.enter.prevent="commitEdit"
          @keydown.esc.prevent.stop="cancelEdit"
          @blur="commitEdit"
        />
        <span v-else class="tab-title">{{ tab.title }}</span>
        <span
          v-if="tabs.length > 1 && tab.id !== editingId"
          class="tab-close"
          @click.stop="emit('close', tab.id)"
        >×</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tabbar {
  display: flex;
  align-items: stretch;
  height: 32px;
  background: var(--ctp-mantle);
  border-bottom: 1px solid var(--ctp-surface0);
  flex: 0 0 auto;
}
.tab-add {
  flex: 0 0 32px;
  border: none;
  border-right: 1px solid var(--ctp-surface0);
  background: transparent;
  color: var(--ctp-subtext0);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.tab-add:hover {
  background: var(--ctp-surface0);
  color: var(--ctp-text);
}
.tab-add-here svg {
  display: block;
}
.tabs-scroll {
  flex: 1 1 0;
  display: flex;
  align-items: stretch;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}
.tabs-scroll::-webkit-scrollbar {
  display: none;
}
.tab {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 10px;
  min-width: 90px;
  max-width: 160px;
  cursor: pointer;
  font-size: 12px;
  color: var(--ctp-subtext0);
  border-right: 1px solid var(--ctp-surface0);
  flex: 0 0 auto;
  user-select: none;
}
.tab:hover {
  background: var(--ctp-surface0);
  color: var(--ctp-text);
}
.tab.active {
  background: var(--ctp-base);
  color: var(--ctp-text);
  border-bottom: 2px solid var(--ctp-blue);
}
.tab-title {
  flex: 1 1 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tab-edit {
  flex: 1 1 0;
  min-width: 0;
  width: 100%;
  border: none;
  outline: 1px solid var(--ctp-blue);
  border-radius: 3px;
  background: var(--ctp-surface0);
  color: var(--ctp-text);
  font-family: inherit;
  font-size: 12px;
  line-height: 1.4;
  padding: 1px 4px;
}
.tab-close {
  flex: 0 0 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 3px;
  font-size: 14px;
  line-height: 1;
  color: var(--ctp-subtext0);
}
.tab-close:hover {
  background: var(--ctp-surface1);
  color: var(--ctp-text);
}
</style>
