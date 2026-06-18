<script setup lang="ts">
import { ref } from 'vue'
import FileTree from './components/FileTree.vue'
import Terminal from './components/Terminal.vue'
import Divider from './components/Divider.vue'

const treeWidth = ref(280)

function onResize(w: number) {
  treeWidth.value = Math.min(500, Math.max(160, w))
}
</script>

<template>
  <div class="app-layout">
    <FileTree :style="{ width: treeWidth + 'px' }" />
    <Divider :currentWidth="treeWidth" @resize="onResize" />
    <Terminal />
  </div>
</template>

<style>
.app-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
</style>
