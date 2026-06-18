<script setup lang="ts">
interface Tab {
  id: string
  title: string
}
const props = defineProps<{ tabs: Tab[]; activeId: string }>()
const emit = defineEmits<{
  add: []
  close: [id: string]
  activate: [id: string]
}>()

function onWheel(e: WheelEvent) {
  e.preventDefault()
  const idx = props.tabs.findIndex((t) => t.id === props.activeId)
  const delta = e.deltaY > 0 ? 1 : -1
  const next = props.tabs[idx + delta]
  if (next) emit('activate', next.id)
}
</script>

<template>
  <div class="tabbar" @wheel="onWheel">
    <div
      v-for="tab in tabs"
      :key="tab.id"
      class="tab"
      :class="{ active: tab.id === activeId }"
      @click="emit('activate', tab.id)"
    >
      <span class="tab-title">{{ tab.title }}</span>
      <span
        v-if="tabs.length > 1"
        class="tab-close"
        @click.stop="emit('close', tab.id)"
      >×</span>
    </div>
    <button class="tab-add" title="新建终端" @click="emit('add')">+</button>
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
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}
.tabbar::-webkit-scrollbar {
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
.tab-add {
  flex: 0 0 32px;
  border: none;
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
</style>
