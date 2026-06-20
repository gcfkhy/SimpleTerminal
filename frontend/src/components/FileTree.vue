<script setup lang="ts">
import { onMounted } from 'vue'
import { useFileTree } from '../composables/useFileTree'
import FileTreeNode from './FileTreeNode.vue'

const { rootNode, currentPath, error, init, goUp, openFolderDialog } = useFileTree()

onMounted(() => {
  void init()
})
</script>

<template>
  <div class="filetree">
    <div class="path-bar">
      <button class="icon-btn" title="上级目录" @click="goUp">⬆</button>
      <span class="path-text" :title="currentPath">{{ currentPath || '…' }}</span>
    </div>

    <div class="list">
      <div v-if="error" class="error">{{ error }}</div>
      <template v-if="rootNode && rootNode.children">
        <FileTreeNode
          v-for="child in rootNode.children"
          :key="child.path"
          :node="child"
          :depth="0"
        />
      </template>
      <div v-else-if="!rootNode && !error" class="empty">
        点击下方「选择目录」，或在终端 cd 进入一个目录
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
  height: 100%;
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
.empty {
  padding: 16px 12px;
  color: var(--ctp-subtext0);
  font-size: 12px;
  line-height: 1.6;
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
</parameter>
