<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { main } from '../../wailsjs/go/models'
import { ReadFileText, ReadFileBase64 } from '../../wailsjs/go/main/App'
import hljs from 'highlight.js'

const props = defineProps<{ file: main.FileEntry }>()
const emit = defineEmits<{ close: [] }>()

const imageExts = new Set(['png','jpg','jpeg','gif','webp','bmp','svg','ico'])
const videoExts = new Set(['mp4','mkv','avi','mov','webm','flv'])
const codeExts  = new Set(['ts','tsx','js','jsx','mjs','vue','json','md','html','htm',
  'css','scss','sass','less','go','py','rs','java','kt','c','h','cpp','cc','hpp','cs',
  'php','rb','swift','dart','lua','sh','bash','zsh','ps1','bat','cmd','yml','yaml',
  'toml','ini','xml','conf','sql'])
const textExts  = new Set(['txt','log','env','csv','tsv'])

const extToLang: Record<string, string> = {
  ts:'typescript', tsx:'typescript', js:'javascript', jsx:'javascript',
  vue:'xml', json:'json', md:'markdown', html:'xml', htm:'xml',
  css:'css', scss:'scss', sass:'scss', less:'less',
  go:'go', py:'python', rs:'rust', java:'java', kt:'kotlin',
  c:'c', h:'c', cpp:'cpp', cc:'cpp', hpp:'cpp', cs:'csharp',
  php:'php', rb:'ruby', swift:'swift', dart:'dart', lua:'lua',
  sh:'bash', bash:'bash', zsh:'bash', ps1:'powershell', bat:'dos', cmd:'dos',
  yml:'yaml', yaml:'yaml', toml:'ini', xml:'xml', sql:'sql',
}

type PType = 'image' | 'video' | 'code' | 'text' | 'unknown'

function getExt(name: string) {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i + 1).toLowerCase() : ''
}

function ptype(file: main.FileEntry): PType {
  const ext = getExt(file.name)
  const low = file.name.toLowerCase()
  if (imageExts.has(ext)) return 'image'
  if (videoExts.has(ext)) return 'video'
  if (codeExts.has(ext)) return 'code'
  if (textExts.has(ext) || ['.gitignore','.env','dockerfile','readme'].includes(low)) return 'text'
  return 'unknown'
}

const kind = computed(() => ptype(props.file))
const loading = ref(false)
const error   = ref('')
const imgSrc  = ref('')
const highlighted = ref('')
const plainText   = ref('')
const videoSrc = computed(() =>
  `/localfile?path=${encodeURIComponent(props.file.path)}`)

watch(() => props.file, async (file) => {
  loading.value = true
  error.value = ''
  imgSrc.value = ''
  highlighted.value = ''
  plainText.value = ''

  try {
    if (kind.value === 'image') {
      imgSrc.value = await ReadFileBase64(file.path)
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
  <div class="preview">
    <div class="preview-header">
      <span class="preview-name" :title="file.path">{{ file.name }}</span>
      <button class="preview-close" @click="emit('close')">×</button>
    </div>

    <div class="preview-body">
      <div v-if="loading"  class="center muted">加载中…</div>
      <div v-else-if="error" class="center err">{{ error }}</div>

      <div v-else-if="kind === 'image'" class="img-wrap">
        <img :src="imgSrc" :alt="file.name" />
      </div>

      <div v-else-if="kind === 'video'" class="video-wrap">
        <video :src="videoSrc" controls preload="metadata" />
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
  height: 100%;
  display: flex;
  flex-direction: column;
  border-left: 1px solid var(--ctp-surface0);
  background: var(--ctp-crust);
  overflow: hidden;
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
}
.img-wrap img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
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

.code-wrap {
  height: 100%;
}
.code-wrap pre {
  margin: 0;
  padding: 8px;
  font-family: 'SF Mono', Consolas, 'MiSans', monospace;
  font-size: 11px;
  line-height: 1.5;
  white-space: pre;
  background: transparent !important;
  color: var(--ctp-text);
}
.code-wrap .hljs {
  background: transparent;
  padding: 0;
}
</style>
