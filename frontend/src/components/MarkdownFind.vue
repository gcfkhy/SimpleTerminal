<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { findMatches, locateOffset, buildTextIndex, MAX_MATCHES } from '../utils/markdown/find'

const props = defineProps<{
  bodyEl: HTMLElement | null
  scrollEl: HTMLElement | null
  open: boolean
}>()
const emit = defineEmits<{ 'update:open': [boolean] }>()

const inputEl = ref<HTMLInputElement | null>(null)
const query = ref('')
const caseSensitive = ref(false)
const countText = ref('')
const countNone = ref(false)
const matchCount = ref(0) // 响应式镜像 ranges.length，供模板按钮禁用态

// CSS Custom Highlight API（常青 WebView2 必有）。用 any 规避 TS lib 尚未含 Highlight 类型。
const HighlightCtor = (window as unknown as { Highlight?: new () => unknown }).Highlight
const cssHighlights = (CSS as unknown as { highlights?: Map<string, unknown> }).highlights
const supportsHighlight = typeof HighlightCtor === 'function' && !!cssHighlights
let hlAll: { add(r: Range): void; clear(): void } | null = null
let hlCur: { add(r: Range): void; clear(): void; priority?: number } | null = null

let ranges: Range[] = []
let cur = -1
let truncated = false
let debounceTimer: ReturnType<typeof setTimeout> | undefined

function ensureHighlights() {
  if (!supportsHighlight || hlAll || !HighlightCtor || !cssHighlights) return
  hlAll = new HighlightCtor() as typeof hlAll
  hlCur = new HighlightCtor() as typeof hlCur
  if (hlCur) hlCur.priority = 1 // 当前项盖在全部匹配之上
  cssHighlights.set('md-find', hlAll as unknown as object)
  cssHighlights.set('md-find-current', hlCur as unknown as object)
}
function clearHighlights() {
  hlAll?.clear()
  hlCur?.clear()
}

function runSearch(scrollToFirst: boolean) {
  ensureHighlights()
  clearHighlights()
  ranges = []
  cur = -1
  truncated = false
  const q = query.value
  if (q && props.bodyEl) {
    const idx = buildTextIndex(props.bodyEl)
    const hits = findMatches(idx.full, q, caseSensitive.value)
    truncated = hits.length >= MAX_MATCHES
    for (const h of hits) {
      const s = locateOffset(idx.segs, h.start, false)
      const e = locateOffset(idx.segs, h.end, true)
      if (!s || !e) continue
      const r = document.createRange()
      try {
        r.setStart(s.node, s.offset)
        r.setEnd(e.node, e.offset)
      } catch {
        continue
      }
      ranges.push(r)
      hlAll?.add(r)
    }
    if (ranges.length) cur = 0
  }
  renderCurrent(scrollToFirst)
  updateCount()
}

function renderCurrent(doScroll: boolean) {
  hlCur?.clear()
  if (cur >= 0 && cur < ranges.length) {
    const r = ranges[cur]
    hlCur?.add(r)
    if (doScroll) scrollRangeIntoView(r)
  }
}

function scrollRangeIntoView(r: Range) {
  const rect = r.getBoundingClientRect()
  if (!rect || (rect.width === 0 && rect.height === 0)) return
  const scroller = props.scrollEl
  const vTop = scroller ? scroller.getBoundingClientRect().top : 0
  const vh = scroller ? scroller.clientHeight : window.innerHeight || document.documentElement.clientHeight
  if (rect.top >= vTop + 60 && rect.bottom <= vTop + vh - 60) return
  const el = r.startContainer.nodeType === 3 ? (r.startContainer as Text).parentElement : (r.startContainer as Element)
  if (el && el.scrollIntoView) el.scrollIntoView({ block: 'center', behavior: 'smooth' })
}

function updateCount() {
  matchCount.value = ranges.length
  if (!query.value) {
    countText.value = ''
    countNone.value = false
  } else if (!ranges.length) {
    countText.value = '无结果'
    countNone.value = true
  } else {
    countText.value = cur + 1 + ' / ' + ranges.length + (truncated ? '+' : '')
    countNone.value = false
  }
}

function go(dir: number) {
  if (!ranges.length) return
  cur = (cur + dir + ranges.length) % ranges.length
  renderCurrent(true)
  updateCount()
}

let composing = false
function onCompositionStart() {
  composing = true
}
function onCompositionEnd() {
  composing = false
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => runSearch(true), 120)
}
function onInput() {
  if (composing) return
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => runSearch(true), 120)
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    go(e.shiftKey ? -1 : 1)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}
function toggleCase() {
  caseSensitive.value = !caseSensitive.value
  runSearch(true)
  inputEl.value?.focus()
}
function close() {
  emit('update:open', false)
}

// 宿主入口：打开（可预填选区）；已开则直接聚焦+重搜。
function openWith(prefill?: string) {
  if (typeof prefill === 'string' && prefill) query.value = prefill
  if (props.open) {
    nextTick(() => {
      inputEl.value?.focus()
      inputEl.value?.select()
      if (query.value) runSearch(true)
      else updateCount()
    })
  } else {
    emit('update:open', true) // 余下聚焦/搜索交给 watch(open)
  }
}

watch(() => props.open, async (open) => {
  if (open) {
    await nextTick()
    inputEl.value?.focus()
    inputEl.value?.select()
    if (query.value) runSearch(true)
    else updateCount()
  } else {
    clearHighlights()
    ranges = []
    cur = -1
    matchCount.value = 0
  }
})

// bodyEl 可能在首次打开后才由父赋值，或内容重渲后变化；若查找条开着且有词，重跑一次（不滚动）。
watch(() => props.bodyEl, () => {
  if (props.open && query.value) runSearch(false)
})

// 晚到内容（图片/KaTeX/Mermaid 异步渲染）稳定后重跑一次（不滚动，避免跳动）。
function onWindowLoad() {
  if (props.open && query.value) runSearch(false)
}
onMounted(() => window.addEventListener('load', onWindowLoad))
onBeforeUnmount(() => {
  clearTimeout(debounceTimer)
  window.removeEventListener('load', onWindowLoad)
  clearHighlights()
})

defineExpose({ openWith, go })
</script>

<template>
  <div class="md-find" :class="{ open }">
    <input
      ref="inputEl"
      class="md-find-input"
      type="text"
      placeholder="查找"
      spellcheck="false"
      aria-label="查找"
      v-model="query"
      @input="onInput"
      @keydown="onKeydown"
      @compositionstart="onCompositionStart"
      @compositionend="onCompositionEnd"
    />
    <span class="md-find-count" :class="{ none: countNone }">{{ countText }}</span>
    <div class="md-find-btn md-find-case" :class="{ active: caseSensitive }" title="区分大小写" @click="toggleCase">Aa</div>
    <div class="md-find-btn" :class="{ disabled: !matchCount }" title="上一个 (Shift+Enter)" @click="go(-1)">↑</div>
    <div class="md-find-btn" :class="{ disabled: !matchCount }" title="下一个 (Enter)" @click="go(1)">↓</div>
    <div class="md-find-btn" title="关闭 (Esc)" @click="close">✕</div>
  </div>
</template>

<style scoped>
/* 查找条锚定在 .md-preview-root 内，absolute 右上角；z-index 高于大纲面板。 */
.md-find {
  position: absolute; top: 12px; right: 16px; z-index: 18; display: none;
  align-items: center; gap: 4px; padding: 5px 6px;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); border-radius: 8px;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.35);
}
.md-find.open { display: flex; }
.md-find-input {
  width: 180px; box-sizing: border-box; border: 1px solid var(--md-ui-border); border-radius: 5px;
  outline: none; background: var(--md-pre-bg); color: var(--md-fg); font-size: 13px; padding: 4px 8px;
}
.md-find-input:focus { border-color: var(--md-link); }
.md-find-count {
  min-width: 54px; text-align: center; font-size: 12px; color: var(--md-muted);
  user-select: none; white-space: nowrap;
}
.md-find-count.none { color: var(--md-code-fg); }
.md-find-btn {
  width: 26px; height: 26px; display: flex; align-items: center; justify-content: center;
  border-radius: 5px; cursor: pointer; font-size: 14px; color: var(--md-fg); user-select: none; flex: 0 0 auto;
}
.md-find-btn:hover { background: var(--md-pre-bg); }
.md-find-btn.active { background: var(--md-pre-bg); color: var(--md-link); font-weight: 600; }
.md-find-btn.disabled { opacity: 0.35; pointer-events: none; }
.md-find-case { font-size: 12px; font-weight: 600; }
</style>
