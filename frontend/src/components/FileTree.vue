<script setup lang="ts">
import { onMounted } from 'vue'
import { OpenFolderDialog } from '../../wailsjs/go/main/App'
import { useFileTree } from '../composables/useFileTree'
import { getFileIcon } from '../composables/useFileIcon'

const { entries, currentPath, parentDir, loadDir, cdToDir } = useFileTree()

onMounted(() => {
  const saved = localStorage.getItem('lastDir')
  if (saved) loadDir(saved)
})

async function selectFolder() {
  const dir = await OpenFolderDialog()
  if (dir) loadDir(dir)
}
</script>

<template>
  <div class="file-tree">
    <div class="tree-header" :title="currentPath">
      {{ currentPath || '未选择目录' }}
    </div>
    <div class="tree-body">
      <div
        v-if="parentDir"
        class="tree-entry dir"
        @click="loadDir(parentDir!)"
      >
        <img :src="getFileIcon('..', true)" class="icon" alt="" @error="(e) => (e.target as HTMLImageElement).src = getFileIcon('', true)" />
        <span class="name">..</span>
      </div>
      <div
        v-for="entry in entries"
        :key="entry.path"
        class="tree-entry"
        :class="{ dir: entry.isDir }"
        draggable="true"
        @click="entry.isDir ? cdToDir(entry.path) : undefined"
        @dragstart="(e) => e.dataTransfer?.setData('text/path', entry.path)"
      >
        <img
          :src="getFileIcon(entry.name, entry.isDir)"
          class="icon"
          alt=""
          @error="(e) => (e.target as HTMLImageElement).src = `https://cdn.jsdelivr.net/gh/material-extensions/vscode-material-icon-theme@main/icons/${entry.isDir ? 'folder' : 'file'}.svg`"
        />
        <span class="name">{{ entry.name }}</span>
      </div>
    </div>
    <button class="select-btn" @click="selectFolder">选择目录</button>
  </div>
</template>

<style scoped>
.file-tree {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #181825;
  border-right: 1px solid #313244;
  overflow: hidden;
  flex-shrink: 0;
}

.tree-header {
  padding: 8px 12px;
  font-size: 11px;
  color: #a6adc8;
  background: #1e1e2e;
  border-bottom: 1px solid #313244;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 0;
}

.tree-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.tree-entry {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  font-size: 13px;
  color: #cdd6f4;
  white-space: nowrap;
  overflow: hidden;
  cursor: default;
  user-select: none;
}

.tree-entry:hover {
  background: #313244;
}

.tree-entry.dir {
  cursor: pointer;
}

.icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  object-fit: contain;
}

.name {
  overflow: hidden;
  text-overflow: ellipsis;
}

.select-btn {
  margin: 8px;
  padding: 6px 0;
  background: #313244;
  color: #cdd6f4;
  border: 1px solid #45475a;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  flex-shrink: 0;
}

.select-btn:hover {
  background: #45475a;
}
</style>
