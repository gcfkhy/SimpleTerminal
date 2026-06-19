<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import type { main } from '../../wailsjs/go/models'
import { ReadFileText, ReadFileBase64 } from '../../wailsjs/go/main/App'
import hljs from 'highlight.js'
import { marked } from 'marked'

const props = withDefaults(defineProps<{
  file: main.FileEntry
  placement?: 'right' | 'top'
}>(), { placement: 'right' })
const emit = defineEmits<{ close: [] }>()

const imageExts = new Set(['png','jpg','jpeg','gif','webp','bmp','svg','ico'])
const videoExts = new Set(['mp4','mkv','avi','mov','webm','flv'])
const mdExts    = new Set(['md','markdown'])
const codeExts  = new Set(['ts','tsx','js','jsx','mjs','vue','json','html','htm',
  'css','scss','sass','less','go','py','rs','java','kt','c','h','cpp','cc','hpp','cs',
  'php','rb','swift','dart','lua','sh','bash','zsh','ps1','bat','cmd','yml','yaml',
  'toml','ini','xml','conf','sql'])
const textExts  = new Set(['txt','log','env','csv','tsv'])

const extToLang: Record<string, string> = {
  ts:'typescript', tsx:'typescript', js:'javascript', jsx:'javascript',
  vue:'xml', json:'json', html:'xml', htm:'xml',
  css:'css', scss:'scss', sass:'scss', less:'less',
  go:'go', py:'python', rs:'rust', java:'java', kt:'kotlin',
  c:'c', h:'c', cpp:'cpp', cc:'cpp', hpp:'cpp', cs:'csharp',
  php:'php', rb:'ruby', swift:'swift', dart:'dart', lua:'lua',
  sh:'bash', bash:'bash', zsh:'bash', ps1:'powershell', bat:'dos', cmd:'dos',
  yml:'yaml', yaml:'yaml', toml:'ini', xml:'xml', sql:'sql',
}

type PType = 'image' | 'video' | 'markdown' | 'code' | 'text' | 'unknown'

function getExt(name: string) {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i + 1).toLowerCase() : ''
}

function ptype(file: main.FileEntry): PType {
  const ext = getExt(file.name)
  const low = file.name.toLowerCase()
  if (imageExts.has(ext)) return 'image'
  if (videoExts.has(ext)) return 'video'
  if (mdExts.has(ext))    return 'markdown'
  if (codeExts.has(ext)) return 'code'
  if (textExts.has(ext) || ['.gitignore','.env','dockerfile','readme'].includes(low)) return 'text'
  return 'unknown'
}

const kind = computed(() => ptype(props.file))
const loading = ref(false)
const error   = ref('')
const imgSrc   = ref('')
const imgScale = ref(1)
const imgX     = ref(0)
const imgY     = ref(0)
const spaceHeld  = ref(false)
const isPanning  = ref(false)
let panStartX = 0, panStartY = 0, panOriginX = 0, panOriginY = 0

const imgCursor = computed(() => {
  if (isPanning.value) return 'grabbing'
  if (spaceHeld.value) return 'grab'
  return 'default'
})

function onKeyDown(e: KeyboardEvent) {
  if (e.code === 'Space' && kind.value === 'image') {
    e.preventDefault()
    spaceHeld.value = true
  }
}
function onKeyUp(e: KeyboardEvent) {
  if (e.code === 'Space') {
    spaceHeld.value = false
    isPanning.value = false
  }
}
function onImgMouseDown(e: MouseEvent) {
  if (!spaceHeld.value) return
  e.preventDefault()
  isPanning.value = true
  panStartX = e.clientX; panStartY = e.clientY
  panOriginX = imgX.value; panOriginY = imgY.value
}
function onImgMouseMove(e: MouseEvent) {
  if (!isPanning.value) return
  imgX.value = panOriginX + (e.clientX - panStartX)
  imgY.value = panOriginY + (e.clientY - panStartY)
}
function onImgMouseUp() { isPanning.value = false }

onMounted(() => {
  window.addEventListener('keydown', onKeyDown)
  window.addEventListener('keyup', onKeyUp)
  window.addEventListener('mousemove', onImgMouseMove)
  window.addEventListener('mouseup', onImgMouseUp)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
  window.removeEventListener('mousemove', onImgMouseMove)
  window.removeEventListener('mouseup', onImgMouseUp)
})

const highlighted = ref('')
const markdownHtml = ref('')
const plainText   = ref('')
const videoSrc = computed(() =>
  `/localfile?path=${encodeURIComponent(props.file.path)}`)

function onImgWheel(e: WheelEvent) {
  e.preventDefault()
  const factor = e.deltaY < 0 ? 1.1 : 0.9
  imgScale.value = Math.min(10, Math.max(0.1, imgScale.value * factor))
}

watch(() => props.file, async (file) => {
  loading.value = true
  error.value = ''
  imgSrc.value = ''
  imgScale.value = 1
  imgX.value = 0
  imgY.value = 0
  highlighted.value = ''
  markdownHtml.value = ''
  plainText.value = ''

  try {
    if (kind.value === 'image') {
      imgSrc.value = await ReadFileBase64(file.path)
    } else if (kind.value === 'markdown') {
      const raw = await ReadFileText(file.path, 500 * 1024)
      markdownHtml.value = await marked(raw, { async: true })
    } else if (kind.value === 'code' || kind.value === 'text') {
      const raw = await ReadFileText(file.path, 200 * 1024)
      plainText.value = raw
      if (kind.value === 'code') {
        const lang = extToLang[getExt(file.name)]
        try {
          highlighted.value = lang
            ? hljs.highlight(raw, { language: lang }).value
            : hljs.highlightAuto(raw).value
        } catch {
          highlighted.value = hljs.highlightAuto(raw).value
        }
      }
    }
  } catch (e) {
    error.value = String(e)
  }
  loading.value = false
}, { immediate: true })
</script>

<template>
  <div class="preview" :class="placement === 'top' ? 'preview--top' : 'preview--right'">
    <div class="preview-header">
      <span class="preview-name" :title="file.path">{{ file.name }}</span>
      <button class="preview-close" @click="emit('close')">×</button>
    </div>

    <div class="preview-body">
      <div v-if="loading"  class="center muted">加载中…</div>
      <div v-else-if="error" class="center err">{{ error }}</div>

      <div v-else-if="kind === 'image'"
           class="img-wrap"
           :style="{ cursor: imgCursor }"
           @wheel.prevent="onImgWheel"
           @mousedown="onImgMouseDown">
        <img
          :src="imgSrc"
          :alt="file.name"
          :draggable="false"
          :style="{ transform: `translate(${imgX}px, ${imgY}px) scale(${imgScale})` }"
        />
      </div>

      <div v-else-if="kind === 'video'" class="video-wrap">
        <video :src="videoSrc" controls preload="metadata" />
      </div>

      <div v-else-if="kind === 'markdown'" class="md-wrap">
        <div class="md-body" v-html="markdownHtml"></div>
      </div>

      <div v-else-if="kind === 'code'" class="code-wrap">
        <pre><code class="hljs" v-html="highlighted"></code></pre>
      </div>

      <div v-else-if="kind === 'text'" class="code-wrap">
        <pre>{{ plainText }}</pre>
      </div>

      <div v-else class="center muted">
        无法预览 · {{ getExt(file.name) || '未知类型' }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.preview {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  background: var(--ctp-crust);
  overflow: hidden;
}
/* 左右布局：预览栏在右侧，交叉轴（高度）由 flex stretch 撑满 */
.preview--right {
  border-left: 1px solid var(--ctp-surface0);
}
/* 上下布局：预览栏在终端上方，交叉轴（宽度）由 flex stretch 撑满 */
.preview--top {
  border-bottom: 1px solid var(--ctp-surface0);
}

.preview-header {
  flex: 0 0 28px;
  display: flex;
  align-items: center;
  padding: 0 8px;
  gap: 6px;
  border-bottom: 1px solid var(--ctp-surface0);
}
.preview-name {
  flex: 1 1 0;
  font-size: 12px;
  color: var(--ctp-subtext1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.preview-close {
  background: transparent;
  border: none;
  color: var(--ctp-subtext0);
  font-size: 16px;
  cursor: pointer;
  padding: 0 2px;
  line-height: 1;
}
.preview-close:hover { color: var(--ctp-text); }

.preview-body {
  flex: 1 1 0;
  min-height: 0;
  overflow: auto;
}

.center {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 12px;
}
.muted { color: var(--ctp-subtext0); }
.err   { color: #f38ba8; }

.img-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 8px;
  overflow: hidden;
  user-select: none;
}
.img-wrap img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  transform-origin: center center;
  transition: transform 0.08s;
  pointer-events: none;
}

.video-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 4px;
}
.video-wrap video {
  max-width: 100%;
  max-height: 100%;
}

/* Markdown 渲染区 */
.md-wrap {
  padding: 16px 20px;
  height: 100%;
}
.md-body {
  font-family: 'MiSans', 'Segoe UI', sans-serif;
  font-size: 14px;
  line-height: 1.7;
  color: var(--ctp-text);
}
/* 标题 */
.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3),
.md-body :deep(h4),
.md-body :deep(h5),
.md-body :deep(h6) {
  color: var(--ctp-lavender);
  margin: 1.2em 0 0.4em;
  font-weight: 600;
  line-height: 1.3;
}
.md-body :deep(h1) { font-size: 1.6em; border-bottom: 1px solid var(--ctp-surface0); padding-bottom: 0.3em; }
.md-body :deep(h2) { font-size: 1.3em; border-bottom: 1px solid var(--ctp-surface0); padding-bottom: 0.2em; }
.md-body :deep(h3) { font-size: 1.1em; }
/* 段落/列表 */
.md-body :deep(p)  { margin: 0.6em 0; }
.md-body :deep(ul),
.md-body :deep(ol) { padding-left: 1.5em; margin: 0.5em 0; }
.md-body :deep(li) { margin: 0.2em 0; }
/* 链接 */
.md-body :deep(a)  { color: var(--ctp-blue); text-decoration: none; }
.md-body :deep(a:hover) { text-decoration: underline; }
/* 行内代码 */
.md-body :deep(code) {
  font-family: 'SF Mono', Consolas, 'MiSans', monospace;
  font-size: 0.88em;
  font-weight: 500;
  background: var(--ctp-surface0);
  color: #f38ba8;
  padding: 0.1em 0.4em;
  border-radius: 4px;
}
/* 代码块 */
.md-body :deep(pre) {
  background: var(--ctp-mantle);
  border: 1px solid var(--ctp-surface0);
  border-radius: 6px;
  padding: 12px;
  overflow-x: auto;
  margin: 0.8em 0;
}
.md-body :deep(pre code) {
  background: transparent;
  padding: 0;
  font-size: 13px;
  font-family: 'SF Mono', Consolas, 'MiSans', monospace;
  font-weight: 500;
  color: var(--ctp-text);
}
/* 引用 */
.md-body :deep(blockquote) {
  border-left: 3px solid var(--ctp-overlay0);
  margin: 0.8em 0;
  padding: 0.3em 1em;
  color: var(--ctp-subtext0);
  background: var(--ctp-mantle);
  border-radius: 0 4px 4px 0;
}
/* 分隔线 */
.md-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--ctp-surface0);
  margin: 1em 0;
}
/* 表格 */
.md-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.8em 0;
  font-size: 13px;
}
.md-body :deep(th),
.md-body :deep(td) {
  border: 1px solid var(--ctp-surface0);
  padding: 6px 10px;
  text-align: left;
}
.md-body :deep(th) {
  background: var(--ctp-surface0);
  color: var(--ctp-lavender);
}
.md-body :deep(tr:nth-child(even)) {
  background: var(--ctp-mantle);
}
/* 图片 */
.md-body :deep(img) {
  max-width: 100%;
  border-radius: 4px;
}

/* 代码高亮区 */
.code-wrap {
  height: 100%;
}
.code-wrap pre,
.code-wrap pre code {
  font-family: 'SF Mono', Consolas, 'MiSans', monospace !important;
  font-size: 15px !important;
  font-weight: 500 !important;
  line-height: 1.5;
}
.code-wrap pre {
  margin: 0;
  padding: 8px;
  white-space: pre;
  background: transparent !important;
  color: var(--ctp-text);
}
.code-wrap .hljs {
  background: transparent;
  padding: 0;
}
</style>
