<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { extractHeadings, buildOutlineTree, type OutlineNode } from '../utils/markdown/outline'

const props = defineProps<{
  bodyEl: HTMLElement | null
  scrollEl: HTMLElement | null
  html: string
  open: boolean
  mode: 'push' | 'overlay'
  width: number
}>()
const emit = defineEmits<{
  'update:open': [boolean]
  'update:mode': ['push' | 'overlay']
  'update:width': [number]
  available: [boolean]
}>()

// 扁平化的可见项（用 collapsed 集合 + ancestorIds 判断可见性，避免递归子组件）。
interface FlatItem {
  id: string
  text: string
  level: number
  hasChildren: boolean
  ancestorIds: string[]
  el: HTMLElement | null
}

const flat = ref<FlatItem[]>([])
const collapsed = ref<Set<string>>(new Set())
const activeId = ref('')
const panelEl = ref<HTMLElement | null>(null)
const listEl = ref<HTMLElement | null>(null)

function flatten(nodes: OutlineNode[], ancestors: string[], out: FlatItem[]) {
  for (const n of nodes) {
    out.push({
      id: n.id,
      text: n.text,
      level: n.level,
      hasChildren: n.children.length > 0,
      ancestorIds: ancestors,
      el: n.el,
    })
    if (n.children.length) flatten(n.children, ancestors.concat(n.id), out)
  }
}

function rebuild() {
  const items = extractHeadings(props.bodyEl)
  const out: FlatItem[] = []
  flatten(buildOutlineTree(items), [], out)
  flat.value = out
  // 清理已不存在的折叠/活跃 id（内容重渲后）
  const ids = new Set(out.map((i) => i.id))
  const nextCollapsed = new Set<string>()
  collapsed.value.forEach((id) => { if (ids.has(id)) nextCollapsed.add(id) })
  collapsed.value = nextCollapsed
  if (!ids.has(activeId.value)) activeId.value = ''
  emit('available', out.length > 0)
  setupObserver()
}

const visible = computed(() =>
  flat.value.filter((it) => it.ancestorIds.every((a) => !collapsed.value.has(a))),
)
const hasHeadings = computed(() => flat.value.length > 0)

function caretFor(it: FlatItem): string {
  if (!it.hasChildren) return ''
  return collapsed.value.has(it.id) ? '▸' : '▾'
}
function paddingLeft(level: number): string {
  return 4 + (level - 1) * 14 + 'px'
}
function toggleCollapse(id: string) {
  const s = new Set(collapsed.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  collapsed.value = s
}
function onClickHeading(it: FlatItem) {
  const target = it.id ? document.getElementById(it.id) : null
  if (target) target.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// ── 滚动联动高亮 ──
let io: IntersectionObserver | null = null
const inView = new Set<Element>()

function setupObserver() {
  if (io) {
    io.disconnect()
    io = null
  }
  inView.clear()
  if (!flat.value.length) return
  io = new IntersectionObserver(
    (entries) => {
      entries.forEach((en) => {
        if (en.isIntersecting) inView.add(en.target)
        else inView.delete(en.target)
      })
      pickActive()
    },
    { root: props.scrollEl || null, rootMargin: '0px 0px -70% 0px', threshold: 0 },
  )
  flat.value.forEach((it) => {
    if (it.el) io!.observe(it.el)
  })
  pickActive()
}

function pickActive() {
  if (!flat.value.length) return
  if (inView.size) {
    const best = flat.value.find((it) => it.el && inView.has(it.el))
    if (best) {
      setActive(best.id)
      return
    }
  }
  let cur: FlatItem | null = null
  for (const it of flat.value) {
    if (it.el && it.el.getBoundingClientRect().top <= 80) cur = it
    else break
  }
  if (cur) setActive(cur.id)
}

function setActive(id: string) {
  if (id === activeId.value) return
  activeId.value = id
  // 自动展开活跃项被折叠的祖先，避免高亮落在隐藏链接上
  const it = flat.value.find((x) => x.id === id)
  if (it && it.ancestorIds.some((a) => collapsed.value.has(a))) {
    const s = new Set(collapsed.value)
    it.ancestorIds.forEach((a) => s.delete(a))
    collapsed.value = s
  }
  // 把活跃链接滚进列表可视区
  nextTick(() => {
    const a = listEl.value?.querySelector<HTMLElement>('.md-outline-link.active')
    if (a && listEl.value) {
      const r = a.getBoundingClientRect()
      const lr = listEl.value.getBoundingClientRect()
      if (r.top < lr.top || r.bottom > lr.bottom) a.scrollIntoView({ block: 'nearest' })
    }
  })
}

// ── 拖拽调宽 ──
const MINW = 180
const MAXW = 480
let dragging = false
function onGripDown(e: MouseEvent) {
  e.preventDefault()
  dragging = true
  document.body.classList.add('md-outline-resizing')
}
function onMouseMove(e: MouseEvent) {
  if (!dragging || !panelEl.value) return
  const left = panelEl.value.getBoundingClientRect().left
  const w = Math.min(MAXW, Math.max(MINW, e.clientX - left))
  emit('update:width', w)
}
function onMouseUp() {
  if (!dragging) return
  dragging = false
  document.body.classList.remove('md-outline-resizing')
}

// ── 模式 / 开关 ──
function toggleMode() {
  emit('update:mode', props.mode === 'overlay' ? 'push' : 'overlay')
}
function close() {
  emit('update:open', false)
}
function openPanel() {
  emit('update:open', true)
}

// ── 生命周期 / 监听 ──
watch(() => props.html, async () => {
  await nextTick()
  rebuild()
})
watch(() => props.bodyEl, async () => {
  await nextTick()
  rebuild()
})
watch(() => props.scrollEl, (el, old) => {
  if (old) old.removeEventListener('scroll', pickActive)
  if (el) el.addEventListener('scroll', pickActive, { passive: true })
  setupObserver()
})

onMounted(() => {
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
  props.scrollEl?.addEventListener('scroll', pickActive, { passive: true })
  nextTick(rebuild)
})
onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
  props.scrollEl?.removeEventListener('scroll', pickActive)
  if (io) io.disconnect()
})

defineExpose({ hasHeadings })
</script>

<template>
  <!-- 左缘常驻把手：仅大纲关闭且有标题时显示 -->
  <div v-show="!open && hasHeadings" class="md-outline-handle" title="大纲导航" @click="openPanel">☰</div>

  <div
    class="md-outline"
    :class="{ open }"
    ref="panelEl"
    :style="{ width: width + 'px', transform: open ? 'translateX(0)' : 'translateX(-100%)' }"
  >
    <div class="md-outline-header">
      <span class="md-outline-title">大纲</span>
      <div
        class="md-outline-hbtn"
        :title="mode === 'push' ? '当前：推送模式，点击切换浮层' : '当前：浮层模式，点击切换推送'"
        @click="toggleMode"
      >⇆</div>
      <div class="md-outline-hbtn" title="关闭大纲" @click="close">✕</div>
    </div>

    <div class="md-outline-list" ref="listEl">
      <div
        v-for="it in visible"
        :key="it.id"
        class="md-outline-row"
        :style="{ paddingLeft: paddingLeft(it.level) }"
      >
        <span class="md-outline-caret" @click.stop="toggleCollapse(it.id)">{{ caretFor(it) }}</span>
        <a
          class="md-outline-link"
          :class="{ active: it.id === activeId }"
          :href="'#' + it.id"
          @click.prevent="onClickHeading(it)"
        >{{ it.text }}</a>
      </div>
    </div>

    <div class="md-outline-grip" title="拖拽调整宽度" @mousedown="onGripDown"></div>
  </div>
</template>

<style scoped>
/* 面板锚定在 .md-preview-root（position:relative, overflow:hidden）内 —— absolute 而非 fixed，
   不溢出预览面板，且继承 .md-preview-root[data-theme] 上的 --md-* 主题变量。 */
.md-outline-handle {
  position: absolute; left: 0; top: 50%; transform: translateY(-50%);
  width: 18px; height: 64px; display: flex; align-items: center; justify-content: center;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); border-left: none;
  border-radius: 0 8px 8px 0; color: var(--md-fg); font-size: 13px; cursor: pointer;
  z-index: 14; opacity: 0.7; user-select: none;
}
.md-outline-handle:hover { opacity: 1; }

.md-outline {
  position: absolute; left: 0; top: 0; bottom: 0;
  display: flex; flex-direction: column; box-sizing: border-box;
  background: var(--md-ui-bg); border-right: 1px solid var(--md-ui-border); color: var(--md-fg);
  z-index: 13; transition: transform 0.2s ease; box-shadow: 2px 0 16px rgba(0, 0, 0, 0.18);
}
body.md-outline-resizing .md-outline { transition: none; }

.md-outline-header {
  display: flex; align-items: center; gap: 6px; padding: 8px 10px;
  border-bottom: 1px solid var(--md-ui-border); flex: 0 0 auto;
}
.md-outline-title { font-size: 13px; font-weight: 600; flex: 1 1 auto; }
.md-outline-hbtn {
  width: 22px; height: 22px; display: flex; align-items: center; justify-content: center;
  border-radius: 4px; cursor: pointer; font-size: 13px; color: var(--md-muted);
}
.md-outline-hbtn:hover { background: var(--md-pre-bg); color: var(--md-fg); }

.md-outline-list { flex: 1 1 auto; overflow: auto; padding: 6px 4px 12px; }
.md-outline-row { display: flex; align-items: flex-start; gap: 2px; border-radius: 4px; }
.md-outline-row:hover { background: var(--md-pre-bg); }
.md-outline-caret {
  flex: 0 0 auto; width: 14px; text-align: center; font-size: 10px; line-height: 1.9;
  color: var(--md-muted); cursor: pointer; user-select: none;
}
.md-outline-link {
  flex: 1 1 auto; padding: 3px 4px; font-size: 13px; line-height: 1.4; color: var(--md-fg);
  text-decoration: none; cursor: pointer;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.md-outline-link:hover { color: var(--md-link); }
.md-outline-link.active { color: var(--md-link); font-weight: 600; }

.md-outline-grip { position: absolute; top: 0; right: -3px; width: 6px; height: 100%; cursor: col-resize; z-index: 1; }
.md-outline-grip:hover { background: var(--md-link); opacity: 0.4; }
</style>
