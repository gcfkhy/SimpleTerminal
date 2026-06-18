<script setup lang="ts">
import { ref } from 'vue'
import { useTerminal } from '../composables/useTerminal'
import { useDragToTerminal } from '../composables/useDragToTerminal'

const containerRef = ref<HTMLElement>()
const { term } = useTerminal(containerRef)
useDragToTerminal(containerRef, term)
</script>

<template>
  <div ref="containerRef" class="terminal-wrap" />
</template>

<style scoped>
.terminal-wrap {
  flex: 1;
  height: 100vh;
  overflow: hidden;
  background: #1e1e2e;
  padding: 2px;
}
</style>
