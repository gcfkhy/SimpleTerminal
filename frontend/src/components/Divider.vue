<script setup lang="ts">
const props = defineProps<{ currentWidth: number }>()
const emit = defineEmits<{ resize: [width: number] }>()

let startX = 0
let startWidth = 0

function onMousedown(e: MouseEvent) {
  startX = e.clientX
  startWidth = props.currentWidth
  document.addEventListener('mousemove', onMousemove)
  document.addEventListener('mouseup', onMouseup)
}

function onMousemove(e: MouseEvent) {
  emit('resize', startWidth + (e.clientX - startX))
}

function onMouseup() {
  document.removeEventListener('mousemove', onMousemove)
  document.removeEventListener('mouseup', onMouseup)
}
</script>

<template>
  <div class="divider" @mousedown="onMousedown" />
</template>

<style scoped>
.divider {
  width: 4px;
  background: #313244;
  cursor: col-resize;
  flex-shrink: 0;
  transition: background 0.15s;
}
.divider:hover,
.divider:active {
  background: #89b4fa;
}
</style>
