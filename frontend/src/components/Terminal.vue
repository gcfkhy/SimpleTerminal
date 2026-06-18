<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, nextTick } from 'vue'
import { useTerminal } from '../composables/useTerminal'
import { useDragToTerminal } from '../composables/useDragToTerminal'

const props = defineProps<{
  tabId: string
  isActive: boolean
  initialDir?: string
}>()

const containerRef = ref<HTMLDivElement | null>(null)
const { mount, getTerm, dispose, fit } = useTerminal(props.tabId)
let cleanupDrag: (() => void) | null = null

onMounted(() => {
  if (!containerRef.value) return
  mount(containerRef.value, props.initialDir)
  cleanupDrag = useDragToTerminal(containerRef.value, getTerm)
})

// Tab 被切换回来时重新 fit，保证尺寸正确
watch(() => props.isActive, (active) => {
  if (active) {
    void nextTick(() => fit())
  }
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
  width: 100%;
  height: 100%;
  background: var(--ctp-base);
  padding: 6px 0 6px 8px;
  overflow: hidden;
}
.terminal :deep(.xterm) {
  height: 100%;
}
</style>
