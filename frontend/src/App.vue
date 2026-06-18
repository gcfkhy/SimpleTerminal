<script setup lang="ts">
import { ref } from 'vue'
import FileTree from './components/FileTree.vue'
import Divider from './components/Divider.vue'
import Terminal from './components/Terminal.vue'

const MIN_WIDTH = 160
const MAX_WIDTH = 500
const WIDTH_KEY = 'treeWidth'

function clamp(w: number): number {
  return Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, w))
}

// 目录树宽度（160–500px），持久化到 localStorage
const saved = Number(localStorage.getItem(WIDTH_KEY))
const treeWidth = ref<number>(Number.isFinite(saved) && saved > 0 ? clamp(saved) : 280)

function onResize(deltaX: number) {
  treeWidth.value = clamp(treeWidth.value + deltaX)
  localStorage.setItem(WIDTH_KEY, String(treeWidth.value))
}
</script>

<template>
  <div class="layout">
    <div class="tree-pane" :style="{ width: treeWidth + 'px' }">
      <FileTree />
    </div>
    <Divider @resize="onResize" />
    <Terminal />
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
</style>
