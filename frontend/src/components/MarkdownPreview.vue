<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import mermaid from 'mermaid'
import 'katex/dist/katex.min.css'
import '../assets/markdown/themes.css'
import '../assets/markdown/content.css'
import { renderMarkdownToHtml } from '../utils/markdown/render'
import { MARKDOWN_THEMES, DEFAULT_THEME_ID } from '../utils/markdown/themes'
import { BrowserOpenURL } from '../../wailsjs/runtime'
import { buildExportHtml } from '../utils/markdown/export-html'
import { SaveExportFile, ExportPdf, SavePngBase64 } from '../../wailsjs/go/main/App'
import html2canvas from 'html2canvas'
import MarkdownOutline from './MarkdownOutline.vue'
import MarkdownFind from './MarkdownFind.vue'
import '../assets/markdown/find-highlight.css'

const props = defineProps<{ source: string; filePath: string }>()
const emit = defineEmits<{ refresh: [] }>()

const theme = ref(localStorage.getItem('md-preview-theme') || DEFAULT_THEME_ID)
const themeOpen = ref(false)
const exportOpen = ref(false)
const html = computed(() => renderMarkdownToHtml(props.source))
const bodyEl = ref<HTMLElement | null>(null)
const rootEl = ref<HTMLElement | null>(null)
const scrollEl = ref<HTMLElement | null>(null)

// ── 大纲状态（持久化）──
const outlineOpen = ref(localStorage.getItem('md-preview-outline-open') === '1')
const outlineMode = ref<'push' | 'overlay'>(
  localStorage.getItem('md-preview-outline-mode') === 'overlay' ? 'overlay' : 'push',
)
const outlineWidth = ref(parseInt(localStorage.getItem('md-preview-outline-width') || '260', 10) || 260)
const outlineAvailable = ref(false)
function setOutlineOpen(v: boolean) {
  outlineOpen.value = v
  localStorage.setItem('md-preview-outline-open', v ? '1' : '0')
}
function toggleOutline() {
  if (outlineAvailable.value) setOutlineOpen(!outlineOpen.value)
}
function setOutlineMode(m: 'push' | 'overlay') {
  outlineMode.value = m
  localStorage.setItem('md-preview-outline-mode', m)
}
function setOutlineWidth(w: number) {
  outlineWidth.value = w
  localStorage.setItem('md-preview-outline-width', String(w))
}
function onOutlineAvailable(v: boolean) {
  outlineAvailable.value = v
  if (!v && outlineOpen.value) setOutlineOpen(false) // 无标题时强制收起
}

// ── 查找状态（临时）──
const findRef = ref<InstanceType<typeof MarkdownFind> | null>(null)
const findOpen = ref(false)
function openFind(prefill?: string) {
  findRef.value?.openWith(prefill)
}

// Ctrl/⌘+F 仅在「焦点/鼠标在预览内」时打开查找，避免抢终端按键。
const previewHovered = ref(false)
function onGlobalKeydown(e: KeyboardEvent) {
  const inPreview = previewHovered.value || !!rootEl.value?.contains(document.activeElement)
  const mod = e.ctrlKey || e.metaKey
  if (mod && !e.altKey && (e.key === 'f' || e.key === 'F')) {
    if (!inPreview) return
    e.preventDefault()
    e.stopPropagation()
    let sel = ''
    try {
      const s = window.getSelection()
      if (s && !s.isCollapsed) sel = String(s).trim()
    } catch {
      /* noop */
    }
    openFind(sel || undefined)
  } else if (e.key === 'F3' && findOpen.value && inPreview) {
    e.preventDefault()
    findRef.value?.go(e.shiftKey ? -1 : 1)
  }
}

const isDark = computed(
  () => MARKDOWN_THEMES.find(t => t.id === theme.value)?.group !== 'light',
)
const darkThemes = MARKDOWN_THEMES.filter(t => t.group === 'dark')
const lightThemes = MARKDOWN_THEMES.filter(t => t.group === 'light')

function pickTheme(id: string) {
  theme.value = id
  localStorage.setItem('md-preview-theme', id)
  themeOpen.value = false
}

function toggleTheme() {
  themeOpen.value = !themeOpen.value
  exportOpen.value = false
}
function toggleExport() {
  exportOpen.value = !exportOpen.value
  themeOpen.value = false
}

// ── 缩放（正文字号）：➖/➕ 按钮 + Ctrl+滚轮，居中显示比例、点击还原，记忆 ──
const zoom = ref(parseFloat(localStorage.getItem('md-preview-zoom') || '1') || 1)
const bodyFontPx = computed(() => 14 * zoom.value)
const zoomHint = ref('')
let zoomHintTimer: ReturnType<typeof setTimeout> | undefined

function setZoom(z: number, showHint = true) {
  zoom.value = Math.min(3, Math.max(0.3, Math.round(z * 100) / 100))
  localStorage.setItem('md-preview-zoom', String(zoom.value))
  if (showHint) {
    zoomHint.value = Math.round(zoom.value * 100) + '%'
    clearTimeout(zoomHintTimer)
    zoomHintTimer = setTimeout(() => (zoomHint.value = ''), 1400)
  }
}
function zoomIn() { setZoom(zoom.value + 0.1) }
function zoomOut() { setZoom(zoom.value - 0.1) }
function resetZoom() { setZoom(1); zoomHint.value = '' }
function onWheel(e: WheelEvent) {
  if (!e.ctrlKey) return
  e.preventDefault()
  setZoom(zoom.value + (e.deltaY < 0 ? 0.1 : -0.1))
}

function doRefresh() {
  emit('refresh')
  showToast('已刷新')
}

// mermaid：每次内容/主题变化后，对容器内 .mermaid 节点重渲
async function runMermaid() {
  if (!bodyEl.value) return
  const nodes = bodyEl.value.querySelectorAll<HTMLElement>('.mermaid:not([data-processed])')
  if (!nodes.length) return
  mermaid.initialize({
    startOnLoad: false,
    theme: isDark.value ? 'dark' : 'default',
    securityLevel: 'loose',
  })
  try {
    await mermaid.run({ nodes: Array.from(nodes) })
  } catch (e) {
    console.error('mermaid render error:', e)
  }
}

watch([html, theme], async () => {
  await nextTick()
  await runMermaid()
})
onMounted(async () => {
  await nextTick()
  await runMermaid()
  window.addEventListener('keydown', onGlobalKeydown, true)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeydown, true)
})

// 链接：外链走系统浏览器；页内锚点容器内平滑滚动
function onClick(e: MouseEvent) {
  const a = (e.target as HTMLElement).closest('a')
  if (!a) return
  const href = a.getAttribute('href')
  if (!href) return
  if (href.startsWith('#')) {
    e.preventDefault()
    const id = decodeURIComponent(href.slice(1))
    const target = id ? rootEl.value?.querySelector('#' + CSS.escape(id)) : null
    target?.scrollIntoView({ behavior: 'smooth' })
    return
  }
  e.preventDefault()
  BrowserOpenURL(href)
}

const toast = ref('')
let toastTimer: ReturnType<typeof setTimeout> | undefined
function showToast(msg: string) {
  toast.value = msg
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = ''), 2600)
}

function pdfName(ext: string) {
  return (props.filePath.split(/[\\/]/).pop() || 'export').replace(/\.md$/i, '') + ext
}

async function exportHtml() {
  if (!rootEl.value) return
  exportOpen.value = false
  const name = pdfName('.html')
  const content = buildExportHtml(rootEl.value, theme.value, name)
  try {
    const saved = await SaveExportFile(name, content)
    if (saved) showToast('HTML 已导出')
  } catch (e) {
    showToast('HTML 导出失败')
    console.error('导出失败:', e)
  }
}

async function exportPdf() {
  if (!rootEl.value) return
  exportOpen.value = false
  const PAGE_W_IN = 794 / 96 // A4 宽 ≈8.27in（=794px@96dpi，与导出 HTML 同宽）
  const MAX_H_IN = 195 // 单页最大高度，避开 PDF 200in 硬上限；实际高度由 Go 端离屏精确测得
  const name = pdfName('.pdf')
  const content = buildExportHtml(rootEl.value, theme.value, name)
  showToast('正在导出 PDF…')
  try {
    const saved = await ExportPdf(name, content, PAGE_W_IN, MAX_H_IN)
    if (saved) showToast('PDF 已导出')
  } catch (e) {
    showToast('PDF 导出失败: ' + String(e))
    console.error('PDF 导出失败:', e)
  }
}

async function exportPng() {
  if (!rootEl.value) return
  exportOpen.value = false
  showToast('正在导出长图…')
  // 在固定宽度(900px)的离屏容器里渲染当前预览，html2canvas 以 2x 截成清晰长图。
  const bodyHtml = rootEl.value.querySelector('.md-body')?.innerHTML ?? ''
  const probe = document.createElement('div')
  probe.className = 'md-preview-root'
  probe.setAttribute('data-theme', theme.value)
  probe.style.cssText = 'position:fixed;left:-99999px;top:0;width:900px;background:var(--md-bg);'
  probe.innerHTML = `<div class="md-body">${bodyHtml}</div>`
  document.body.appendChild(probe)
  try {
    const bg = getComputedStyle(probe).backgroundColor
    const canvas = await html2canvas(probe, { scale: 2, backgroundColor: bg, useCORS: true })
    const b64 = canvas.toDataURL('image/png').split(',')[1] || ''
    const name = pdfName('.png')
    const saved = await SavePngBase64(name, b64)
    if (saved) showToast('长图已导出')
  } catch (e) {
    showToast('长图导出失败: ' + String(e))
    console.error('长图导出失败:', e)
  } finally {
    document.body.removeChild(probe)
  }
}

defineExpose({ rootEl, bodyEl, scrollEl, theme })
</script>

<template>
  <div class="md-preview-root" :data-theme="theme" ref="rootEl"
       @mouseenter="previewHovered = true" @mouseleave="previewHovered = false">
    <div class="md-scroll" ref="scrollEl" @wheel="onWheel"
         :style="outlineOpen && outlineMode === 'push' ? { paddingLeft: outlineWidth + 'px' } : undefined">
      <div class="md-body" ref="bodyEl" :style="{ fontSize: bodyFontPx + 'px' }"
           v-html="html" @click="onClick"></div>
    </div>

    <MarkdownOutline
      :body-el="bodyEl" :scroll-el="scrollEl" :html="html"
      :open="outlineOpen" :mode="outlineMode" :width="outlineWidth"
      @update:open="setOutlineOpen" @update:mode="setOutlineMode"
      @update:width="setOutlineWidth" @available="onOutlineAvailable" />
    <MarkdownFind
      ref="findRef" :open="findOpen" @update:open="findOpen = $event"
      :body-el="bodyEl" :scroll-el="scrollEl" />

    <!-- 工具条（左→右）：大纲 / 查找 / 主题 / 导出 / 缩小 / 放大 / 刷新 -->
    <div class="md-toolbar">
      <button class="md-tool-btn" title="大纲" :class="{ disabled: !outlineAvailable }" @click="toggleOutline">📑</button>
      <button class="md-tool-btn" title="查找 (Ctrl+F)" @click="openFind()">🔍</button>
      <button class="md-tool-btn" title="主题" @click="toggleTheme">🎨</button>
      <button class="md-tool-btn" title="导出" @click="toggleExport">📤</button>
      <button class="md-tool-btn" title="缩小" @click="zoomOut">➖</button>
      <button class="md-tool-btn" title="放大" @click="zoomIn">➕</button>
      <button class="md-tool-btn" title="刷新" @click="doRefresh">🔄</button>
    </div>

    <div v-if="exportOpen" class="md-panel md-export-panel">
      <div class="md-panel-item" @click="exportPdf">PDF（单页）</div>
      <div class="md-panel-item" @click="exportHtml">HTML</div>
      <div class="md-panel-item" @click="exportPng">长图 (PNG)</div>
    </div>

    <div v-if="themeOpen" class="md-panel md-theme-panel">
      <div class="md-panel-group">暗色</div>
      <div v-for="t in darkThemes" :key="t.id"
           class="md-panel-item" :class="{ active: t.id === theme }"
           @click="pickTheme(t.id)">{{ t.name }}</div>
      <div class="md-panel-group">亮色</div>
      <div v-for="t in lightThemes" :key="t.id"
           class="md-panel-item" :class="{ active: t.id === theme }"
           @click="pickTheme(t.id)">{{ t.name }}</div>
    </div>

    <div v-if="zoomHint" class="md-zoom-hint" title="点击还原 100%" @click="resetZoom">{{ zoomHint }}</div>
    <div v-if="toast" class="md-toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.md-preview-root {
  position: relative;
  height: 100%;
  overflow: hidden;
  background: var(--md-bg);
}
.md-scroll {
  height: 100%;
  overflow: auto;
  transition: padding-left 0.2s ease;
}
.md-toolbar {
  position: absolute; right: 14px; bottom: 14px; display: flex; gap: 8px; z-index: 10;
}
.md-tool-btn {
  width: 34px; height: 34px; border-radius: 50%;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); color: var(--md-fg);
  font-size: 15px; cursor: pointer; opacity: 0.85;
  display: flex; align-items: center; justify-content: center; padding: 0;
}
.md-tool-btn:hover { opacity: 1; }
.md-tool-btn.disabled { opacity: 0.4; pointer-events: none; }
.md-panel {
  position: absolute; right: 14px; bottom: 56px; max-height: 60vh; overflow: auto;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); border-radius: 8px;
  padding: 6px; z-index: 11; min-width: 150px; box-shadow: 0 6px 24px rgba(0,0,0,0.35);
}
.md-panel-group { font-size: 11px; color: var(--md-muted); margin: 6px 6px 2px; }
.md-panel-item { padding: 5px 10px; border-radius: 4px; cursor: pointer; font-size: 13px; color: var(--md-fg); white-space: nowrap; }
.md-panel-item:hover { background: var(--md-pre-bg); }
.md-panel-item.active { background: var(--md-pre-bg); font-weight: 600; }
.md-panel-item.active::after { content: ' ✓'; color: var(--md-link); }
.md-zoom-hint {
  position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  padding: 8px 18px; border-radius: 10px; font-size: 22px; font-weight: 600; line-height: 1;
  background: var(--md-ui-bg); color: var(--md-fg); border: 1px solid var(--md-ui-border);
  box-shadow: 0 6px 24px rgba(0,0,0,0.35); z-index: 19; cursor: pointer; user-select: none;
}
.md-toast {
  position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  padding: 10px 20px; border-radius: 10px; font-size: 14px;
  max-width: 80%; text-align: center; word-break: break-word;
  background: var(--md-ui-bg); color: var(--md-fg); border: 1px solid var(--md-ui-border);
  box-shadow: 0 6px 24px rgba(0,0,0,0.35); z-index: 20;
}
</style>
