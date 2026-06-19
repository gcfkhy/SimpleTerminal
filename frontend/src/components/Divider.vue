<script setup lang="ts">
// 4px 可拖拽分隔线，向父组件发出每次移动的增量。
// orientation = 'vertical'   → 竖向分隔条，col-resize，发出 X 轴增量（左右布局用）
// orientation = 'horizontal' → 横向分隔条，row-resize，发出 Y 轴增量（上下布局用）
const props = withDefaults(defineProps<{ orientation?: 'vertical' | 'horizontal' }>(), {
  orientation: 'vertical',
})
const emit = defineEmits<{ (e: 'resize', delta: number): void }>()

let dragging = false
let last = 0

function pos(e: MouseEvent) {
  return props.orientation === 'vertical' ? e.clientX : e.clientY
}

function onMouseDown(e: MouseEvent) {
  dragging = true
  last = pos(e)
  document.body.style.cursor = props.orientation === 'vertical' ? 'col-resize' : 'row-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e: MouseEvent) {
  if (!dragging) return
  const cur = pos(e)
  const delta = cur - last
  last = cur
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
  <div class="divider" :class="orientation" @mousedown="onMouseDown"></div>
</template>

<style scoped>
.divider {
  flex: 0 0 4px;
  background: var(--ctp-surface0);
  transition: background 0.15s ease;
}
.divider.vertical {
  width: 4px;
  height: 100%;
  cursor: col-resize;
}
.divider.horizontal {
  width: 100%;
  height: 4px;
  cursor: row-resize;
}
.divider:hover {
  background: var(--ctp-blue);
}
</style>
