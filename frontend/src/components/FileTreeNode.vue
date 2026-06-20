<script setup lang="ts">
import { computed } from 'vue'
import { useFileTree, type TreeNode } from '../composables/useFileTree'
import { useFileIcon } from '../composables/useFileIcon'

// 递归组件需要显式命名才能在自身模板里引用。
defineOptions({ name: 'FileTreeNode' })

const props = defineProps<{ node: TreeNode; depth: number }>()
const { open, selectedPath } = useFileTree()
const { getIcon, fallback } = useFileIcon()

const INDENT = 8 // 每层缩进像素（贴近 VSCode）
const BASE = 6 // 第一层左内边距

// 行左内边距 + 缩进参考线：为每个祖先层级画一条淡竖线（仿 VSCode indent guides）。
const rowStyle = computed(() => {
  const style: Record<string, string> = { paddingLeft: props.depth * INDENT + BASE + 'px' }
  if (props.depth > 0) {
    const guides: string[] = []
    for (let i = 0; i < props.depth; i++) {
      const x = i * INDENT + BASE + 7 // 对齐到该层箭头中心附近
      guides.push(
        `linear-gradient(to right, transparent ${x}px, var(--guide) ${x}px, var(--guide) ${x + 1}px, transparent ${x + 1}px)`
      )
    }
    style.backgroundImage = guides.join(',')
  }
  return style
})

function onDragStart(e: DragEvent) {
  if (!e.dataTransfer) return
  e.dataTransfer.setData('text/path', props.node.path)
  e.dataTransfer.setData('text/plain', props.node.path)
  e.dataTransfer.effectAllowed = 'copy'
}

function onIconError(e: Event) {
  const img = e.target as HTMLImageElement
  // 避免回退图标再次出错导致死循环
  if (img.dataset.fallback) return
  img.dataset.fallback = '1'
  img.src = fallback(props.node.isDir)
}
</script>

<template>
  <div
    class="node"
    :class="{ dir: node.isDir, selected: selectedPath === node.path }"
    :style="rowStyle"
    draggable="true"
    :title="node.name"
    @click="open(node)"
    @dragstart="onDragStart"
  >
    <span class="twisty">
      <span v-if="node.isDir" class="chevron" :class="{ open: node.expanded }"></span>
    </span>
    <img
      class="icon"
      :src="getIcon(node.name, node.isDir)"
      alt=""
      draggable="false"
      @error="onIconError"
    />
    <span class="name">{{ node.name }}</span>
  </div>

  <template v-if="node.isDir && node.expanded && node.children">
    <FileTreeNode
      v-for="child in node.children"
      :key="child.path"
      :node="child"
      :depth="depth + 1"
    />
  </template>
</template>

<style scoped>
/* 每行：VSCode 标准 22px 紧凑行高 */
.node {
  display: flex;
  align-items: center;
  height: 22px;
  padding-right: 8px;
  cursor: pointer;
  white-space: nowrap;
  font-size: 13px;
  color: var(--ctp-text);
  /* 缩进参考线颜色 */
  --guide: rgba(166, 173, 200, 0.14);
}
.node:hover {
  background-color: rgba(255, 255, 255, 0.045);
}
.node.selected {
  /* Catppuccin blue 半透明，仿 VSCode 选中高亮 */
  background-color: rgba(137, 180, 250, 0.16);
}

/* 展开箭头列：宽 16px，居中放细 chevron */
.twisty {
  width: 16px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
}
/* 细线 chevron：折叠时朝右 ›，展开时朝下 ⌄ */
.chevron {
  width: 5px;
  height: 5px;
  margin-left: -1px;
  border-right: 1.4px solid var(--ctp-subtext0);
  border-bottom: 1.4px solid var(--ctp-subtext0);
  transform: rotate(-45deg);
  transition: transform 0.12s ease;
}
.chevron.open {
  transform: rotate(45deg);
}

.icon {
  width: 16px;
  height: 16px;
  flex: 0 0 auto;
  margin-right: 6px;
  pointer-events: none;
}
.name {
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
</parameter>
