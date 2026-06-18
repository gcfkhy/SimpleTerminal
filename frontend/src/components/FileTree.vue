<script setup lang="ts">
import { onMounted } from 'vue'
import { useFileTree } from '../composables/useFileTree'
import { useFileIcon } from '../composables/useFileIcon'
import type { main } from '../../wailsjs/go/models'

const { currentPath, entries, error, init, goUp, openFolderDialog, open } = useFileTree()
const { getIcon, fallback } = useFileIcon()

onMounted(() => {
  void init()
})

function onDragStart(e: DragEvent, entry: main.FileEntry) {
  if (!e.dataTransfer) return
  e.dataTransfer.setData('text/path', entry.path)
  e.dataTransfer.setData('text/plain', entry.path)
  e.dataTransfer.effectAllowed = 'copy'
}

function onIconError(e: Event, isDir: boolean) {
  const img = e.target as HTMLImageElement
  // 避免回退图标再次出错导致死循环
  if (img.dataset.fallback) return
  img.dataset.fallback = '1'
  img.src = fallback(isDir)
}
</script>

<template>
  <div class="filetree">
    <div class="path-bar">
      <button class="icon-btn" title="上级目录" @click="goUp">⬆</button>
      <span class="path-text" :title="currentPath">{{ currentPath || '…' }}</span>
    </div>

    <div class="list">
      <div v-if="error" class="error">{{ error }}</div>
      <div
        v-for="entry in entries"
        :key="entry.path"
        class="item"
        :class="{ dir: entry.isDir }"
        draggable="true"
        :title="entry.name"
        @click="open(entry)"
        @dragstart="onDragStart($event, entry)"
      >
        <img
          class="icon"
          :src="getIcon(entry.name, entry.isDir)"
          alt=""
          draggable="false"
          @error="onIconError($event, entry.isDir)"
        />
        <span class="name">{{ entry.name }}</span>
      </div>
    </div>

    <div class="footer">
      <button class="pick-btn" @click="openFolderDialog">选择目录</button>
    </div>
  </div>
</template>

<style scoped>
.filetree {
  display: flex;
  flex-direction: column;
  flex: 1 1 0;
  min-height: 0;
  background: var(--ctp-mantle);
  user-select: none;
}

.path-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--ctp-surface0);
  flex: 0 0 auto;
}
.icon-btn {
  flex: 0 0 auto;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: var(--ctp-surface0);
  color: var(--ctp-text);
  cursor: pointer;
  font-size: 13px;
}
.icon-btn:hover {
  background: var(--ctp-surface1);
}
.path-text {
  flex: 1 1 0;
  min-width: 0;
  font-size: 12px;
  color: var(--ctp-subtext0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}

.list {
  flex: 1 1 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 4px 0;
}
.error {
  padding: 8px;
  color: #f38ba8;
  font-size: 12px;
  word-break: break-all;
}
.item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  cursor: pointer;
  white-space: nowrap;
}
.item:hover {
  background: var(--ctp-surface0);
}
.icon {
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  pointer-events: none;
}
.name {
  font-size: 13px;
  color: var(--ctp-text);
  overflow: hidden;
  text-overflow: ellipsis;
}
.item.dir .name {
  color: var(--ctp-lavender);
}

.footer {
  flex: 0 0 auto;
  padding: 8px;
  border-top: 1px solid var(--ctp-surface0);
}
.pick-btn {
  width: 100%;
  padding: 7px 0;
  border: 1px solid var(--ctp-overlay0);
  border-radius: 6px;
  background: transparent;
  color: var(--ctp-subtext1);
  font-size: 13px;
  font-weight: 400;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.pick-btn:hover {
  background: var(--ctp-surface1);
  color: var(--ctp-text);
}
</style>
