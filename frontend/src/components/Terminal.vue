<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useTerminal } from '../composables/useTerminal'
import { useDragToTerminal } from '../composables/useDragToTerminal'

const containerRef = ref<HTMLDivElement | null>(null)
const { mount, getTerm, dispose } = useTerminal()
let cleanupDrag: (() => void) | null = null

onMounted(() => {
  if (!containerRef.value) return
  mount(containerRef.value)
  cleanupDrag = useDragToTerminal(containerRef.value, getTerm)
})

onBeforeUnmount(() => {
  cleanupDrag?.()
  dispose()
})
</script>

<template>
  <div ref="containerRef" class="terminal"></div>
</template>

<style scoped>
.terminal {
  flex: 1 1 0;
  min-width: 0;
  height: 100%;
  background: var(--ctp-base);
  padding: 6px 0 6px 8px;
  overflow: hidden;
}
/* 让 xterm 填满容器 */
.terminal :deep(.xterm) {
  height: 100%;
}
</style>
