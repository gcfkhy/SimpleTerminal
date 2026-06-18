<script setup lang="ts">
// 4px 可拖拽分隔线，向父组件发出每次移动的水平增量
const emit = defineEmits<{ (e: 'resize', deltaX: number): void }>()

let dragging = false
let lastX = 0

function onMouseDown(e: MouseEvent) {
  dragging = true
  lastX = e.clientX
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e: MouseEvent) {
  if (!dragging) return
  const delta = e.clientX - lastX
  lastX = e.clientX
  emit('resize', delta)
}

function onMouseUp() {
  dragging = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
}
</script>

<template>
  <div class="divider" @mousedown="onMouseDown"></div>
</template>

<style scoped>
.divider {
  flex: 0 0 4px;
  width: 4px;
  height: 100%;
  cursor: col-resize;
  background: var(--ctp-surface0);
  transition: background 0.15s ease;
}
.divider:hover {
  background: var(--ctp-blue);
}
</style>
