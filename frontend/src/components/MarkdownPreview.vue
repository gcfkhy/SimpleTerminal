<script setup lang="ts">
import { ref, watch, computed, onMounted, nextTick } from 'vue'
import mermaid from 'mermaid'
import 'katex/dist/katex.min.css'
import '../assets/markdown/themes.css'
import '../assets/markdown/content.css'
import { renderMarkdownToHtml } from '../utils/markdown/render'
import { MARKDOWN_THEMES, DEFAULT_THEME_ID } from '../utils/markdown/themes'
import { BrowserOpenURL } from '../../wailsjs/runtime'
import { buildExportHtml } from '../utils/markdown/export-html'
import { SaveExportFile, ExportPdf } from '../../wailsjs/go/main/App'

const props = defineProps<{ source: string; filePath: string }>()

const theme = ref(localStorage.getItem('md-preview-theme') || DEFAULT_THEME_ID)
const themeOpen = ref(false)
const html = computed(() => renderMarkdownToHtml(props.source))
const bodyEl = ref<HTMLElement | null>(null)
const rootEl = ref<HTMLElement | null>(null)

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

async function exportHtml() {
  if (!rootEl.value) return
  const name = (props.filePath.split(/[\\/]/).pop() || 'export').replace(/\.md$/i, '') + '.html'
  const content = buildExportHtml(rootEl.value, theme.value, name)
  try {
    const saved = await SaveExportFile(name, content)
    if (saved) showToast('HTML 已导出')
  } catch (e) {
    showToast('HTML 导出失败')
    console.error('导出失败:', e)
  }
}

const toast = ref('')
let toastTimer: ReturnType<typeof setTimeout> | undefined
function showToast(msg: string) {
  toast.value = msg
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = ''), 2600)
}

async function exportPdf() {
  if (!rootEl.value) return
  const PAGE_W_IN = 794 / 96 // A4 宽 ≈8.27in（=794px@96dpi，与导出 HTML 同宽）
  const MAX_H_IN = 195 // 单页最大高度，避开 PDF 200in 硬上限；实际内容高度由 Go 端离屏精确测得
  const name = (props.filePath.split(/[\\/]/).pop() || 'export').replace(/\.md$/i, '') + '.pdf'
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

defineExpose({ rootEl, bodyEl, theme })
</script>

<template>
  <div class="md-preview-root" :data-theme="theme" ref="rootEl">
    <div class="md-scroll">
      <div class="md-body" ref="bodyEl" v-html="html" @click="onClick"></div>
    </div>

    <div class="md-toolbar">
      <button class="md-tool-btn" title="导出 HTML" @click="exportHtml">⬇</button>
      <button class="md-tool-btn" title="导出 PDF" @click="exportPdf">📄</button>
      <button class="md-tool-btn" title="主题" @click="themeOpen = !themeOpen">🎨</button>
    </div>

    <div v-if="toast" class="md-toast">{{ toast }}</div>
    <div v-if="themeOpen" class="md-theme-panel">
      <div class="md-theme-group">暗色</div>
      <div v-for="t in darkThemes" :key="t.id"
           class="md-theme-item" :class="{ active: t.id === theme }"
           @click="pickTheme(t.id)">{{ t.name }}</div>
      <div class="md-theme-group">亮色</div>
      <div v-for="t in lightThemes" :key="t.id"
           class="md-theme-item" :class="{ active: t.id === theme }"
           @click="pickTheme(t.id)">{{ t.name }}</div>
    </div>
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
}
.md-toolbar {
  position: absolute; right: 14px; bottom: 14px; display: flex; gap: 8px; z-index: 10;
}
.md-tool-btn {
  width: 34px; height: 34px; border-radius: 50%;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); color: var(--md-fg);
  font-size: 16px; cursor: pointer; opacity: 0.85;
}
.md-tool-btn:hover { opacity: 1; }
.md-theme-panel {
  position: absolute; right: 14px; bottom: 56px; max-height: 60vh; overflow: auto;
  background: var(--md-ui-bg); border: 1px solid var(--md-ui-border); border-radius: 8px;
  padding: 6px; z-index: 11; min-width: 170px; box-shadow: 0 6px 24px rgba(0,0,0,0.35);
}
.md-theme-group { font-size: 11px; color: var(--md-muted); margin: 6px 6px 2px; }
.md-theme-item { padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 13px; color: var(--md-fg); white-space: nowrap; }
.md-theme-item:hover { background: var(--md-pre-bg); }
.md-theme-item.active { background: var(--md-pre-bg); font-weight: 600; }
.md-theme-item.active::after { content: ' ✓'; color: var(--md-link); }
.md-toast {
  position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  padding: 10px 20px; border-radius: 10px; font-size: 14px;
  max-width: 80%; text-align: center; word-break: break-word;
  background: var(--md-ui-bg); color: var(--md-fg); border: 1px solid var(--md-ui-border);
  box-shadow: 0 6px 24px rgba(0,0,0,0.35); z-index: 20;
}
</style>
