# Markdown 预览大纲目录 + Ctrl+F 查找 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 `MarkdownPreview.vue` 实时预览补齐 vscode-office 同款「左侧大纲目录」与「Ctrl+F 预览内查找」，仅作用于预览、不动导出。

**Architecture:** hybrid —— 把 vscode-office 经过验证的纯函数（建树 / 匹配 / 偏移定位）移植为 TypeScript 工具模块并单测；UI 用两个原生 Vue 组件（`MarkdownOutline.vue` / `MarkdownFind.vue`）实现，接入既有 `bodyEl`/`scrollEl` 引用与 `localStorage`；`MarkdownPreview.vue` 作为宿主挂载二者、加工具栏按钮、持有大纲状态、转发 Ctrl+F。

**Tech Stack:** Vue 3 `<script setup>` + TypeScript、Vite、Vitest(jsdom)、CSS Custom Highlight API、IntersectionObserver。

## Global Constraints

- 仅 Windows / 常青 WebView2 运行环境——CSS Custom Highlight API（`CSS.highlights` / `Highlight`）视为可用，**不实现旧版浮层降级**。
- **不新增第三方依赖**（不引入 `@vue/test-utils` 等）。
- 持久化一律走 `localStorage`，键名沿用 `md-preview-` 前缀：`md-preview-outline-open`（`'0'`/`'1'`）、`md-preview-outline-mode`（`'push'`/`'overlay'`）、`md-preview-outline-width`（px 字符串）。查找状态不持久化。
- 导出三路（HTML / PDF / 长图）保持不变。
- 自动生成目录 `frontend/wailsjs/` 严禁手改。
- 类型校验以 `cd frontend && npm run build`（= `vue-tsc --noEmit && vite build`）为准；单测 `cd frontend && npm run test`。
- 提交信息用中文，结尾固定加：
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
- 全部命令在仓库根 `D:\E\Self\SimpleTerminal` 下执行；涉及前端的 `npx`/`npm` 先 `cd frontend`。

---

### Task 1: 大纲纯逻辑模块 `outline.ts`（TDD）

**Files:**
- Create: `frontend/src/utils/markdown/outline.ts`
- Test: `frontend/src/__tests__/outline.test.ts`

**Interfaces:**
- Consumes: 无（叶子模块）。
- Produces:
  - `interface OutlineItem { level: number; text: string; id: string; el: HTMLElement }`
  - `interface OutlineNode { level: number; text: string; id: string; el: HTMLElement | null; children: OutlineNode[] }`
  - `extractHeadings(bodyEl: HTMLElement | null): OutlineItem[]`
  - `buildOutlineTree(items: OutlineItem[]): OutlineNode[]`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/__tests__/outline.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { extractHeadings, buildOutlineTree, type OutlineItem } from '../utils/markdown/outline'

function h(level: number, text: string, id: string): OutlineItem {
  const el = document.createElement('h' + level)
  el.id = id
  el.textContent = text
  return { level, text, id, el }
}

describe('buildOutlineTree', () => {
  it('常规层级构建嵌套树', () => {
    const tree = buildOutlineTree([h(1, 'A', 'a'), h(2, 'B', 'b'), h(2, 'C', 'c'), h(3, 'D', 'd')])
    expect(tree.length).toBe(1)
    expect(tree[0].id).toBe('a')
    expect(tree[0].children.map((n) => n.id)).toEqual(['b', 'c'])
    expect(tree[0].children[1].children.map((n) => n.id)).toEqual(['d'])
  })

  it('跳级 h1→h3 时 h3 仍挂到最近的更浅标题下', () => {
    const tree = buildOutlineTree([h(1, 'A', 'a'), h(3, 'C', 'c')])
    expect(tree.length).toBe(1)
    expect(tree[0].children.map((n) => n.id)).toEqual(['c'])
  })

  it('空输入返回空数组', () => {
    expect(buildOutlineTree([])).toEqual([])
  })
})

describe('extractHeadings', () => {
  it('抠出 h1..h6 并排除无 id 或空文本的标题', () => {
    const body = document.createElement('div')
    body.innerHTML =
      '<h1 id="a">A</h1>' +
      '<h2>无 id</h2>' +          // 排除
      '<h2 id="blank">   </h2>' + // 排除（空文本）
      '<h3 id="b">B</h3>'
    const items = extractHeadings(body)
    expect(items.map((i) => i.id)).toEqual(['a', 'b'])
    expect(items[0].level).toBe(1)
    expect(items[1].level).toBe(3)
  })

  it('bodyEl 为 null 返回空数组', () => {
    expect(extractHeadings(null)).toEqual([])
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/__tests__/outline.test.ts`
Expected: FAIL（`Failed to resolve import '../utils/markdown/outline'`）。

- [ ] **Step 3: 写实现**

创建 `frontend/src/utils/markdown/outline.ts`：

```ts
// 从 vscode-office resource/markdown/outline.js 移植纯逻辑，适配 TypeScript + SimpleTerminal。

export interface OutlineItem {
  level: number
  text: string
  id: string
  el: HTMLElement
}

export interface OutlineNode {
  level: number
  text: string
  id: string
  el: HTMLElement | null
  children: OutlineNode[]
}

/**
 * 扫描 .md-body 内的 h1..h6 → 扁平项；过滤掉无文本或无 id 的标题
 * （无 id 的标题无法跳转/高亮）。
 */
export function extractHeadings(bodyEl: HTMLElement | null): OutlineItem[] {
  if (!bodyEl) return []
  const hs = Array.from(bodyEl.querySelectorAll<HTMLElement>('h1,h2,h3,h4,h5,h6'))
  return hs
    .map((el) => ({
      level: parseInt(el.tagName.charAt(1), 10),
      text: (el.textContent || '').trim(),
      id: el.id,
      el,
    }))
    .filter((it) => it.text && it.id)
}

/**
 * 扁平标题列表 → 嵌套树。用栈维护祖先链：弹出所有 level >= 当前的栈顶，
 * 余下栈顶即父；空栈则作顶层。跳级（h1→h3）时 h3 仍挂到最近的更浅标题下。
 */
export function buildOutlineTree(items: OutlineItem[]): OutlineNode[] {
  const roots: OutlineNode[] = []
  const stack: OutlineNode[] = []
  ;(items || []).forEach((raw) => {
    const node: OutlineNode = {
      level: raw.level,
      text: raw.text,
      id: raw.id,
      el: raw.el ?? null,
      children: [],
    }
    while (stack.length && stack[stack.length - 1].level >= node.level) stack.pop()
    if (stack.length) stack[stack.length - 1].children.push(node)
    else roots.push(node)
    stack.push(node)
  })
  return roots
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd frontend && npx vitest run src/__tests__/outline.test.ts`
Expected: PASS（5 个用例全绿）。

- [ ] **Step 5: 提交**

```bash
git add frontend/src/utils/markdown/outline.ts frontend/src/__tests__/outline.test.ts
git commit -m "feat: 大纲纯逻辑 extractHeadings/buildOutlineTree + 单测

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: 查找纯逻辑模块 `find.ts`（TDD）

**Files:**
- Create: `frontend/src/utils/markdown/find.ts`
- Test: `frontend/src/__tests__/find.test.ts`

**Interfaces:**
- Consumes: 无。
- Produces:
  - `interface MatchRange { start: number; end: number }`
  - `interface TextSeg { node: Text; start: number }`
  - `interface TextIndex { full: string; segs: TextSeg[] }`
  - `const MAX_MATCHES = 5000`
  - `findMatches(full: string, query: string, caseSensitive: boolean): MatchRange[]`
  - `locateOffset(segs: TextSeg[], pos: number, atEnd: boolean): { node: Text; offset: number } | null`
  - `buildTextIndex(bodyEl: HTMLElement | null): TextIndex`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/__tests__/find.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { findMatches, locateOffset, buildTextIndex, MAX_MATCHES, type TextSeg } from '../utils/markdown/find'

describe('findMatches', () => {
  it('不区分大小写（快路径）找出全部不重叠匹配', () => {
    const m = findMatches('Hello hello HELLO', 'hello', false)
    expect(m.map((x) => x.start)).toEqual([0, 6, 12])
    expect(m[0]).toEqual({ start: 0, end: 5 })
  })

  it('区分大小写只匹配同形', () => {
    const m = findMatches('Hello hello', 'hello', true)
    expect(m).toEqual([{ start: 6, end: 11 }])
  })

  it('空串/空查询返回空', () => {
    expect(findMatches('', 'x', false)).toEqual([])
    expect(findMatches('abc', '', false)).toEqual([])
  })

  it('Unicode 折叠改变长度时走慢路径，命中映射回原串偏移', () => {
    // 'İ'(U+0130).toLowerCase() 长度变 2，触发慢路径
    const m = findMatches('aİb', 'b', false)
    expect(m).toEqual([{ start: 2, end: 3 }])
  })

  it('命中数达 MAX_MATCHES 截断', () => {
    const m = findMatches('a'.repeat(MAX_MATCHES + 1000), 'a', true)
    expect(m.length).toBe(MAX_MATCHES)
  })
})

describe('locateOffset', () => {
  function segOf(text: string, start: number): TextSeg {
    return { node: document.createTextNode(text), start }
  }
  const segs = [segOf('abc', 0), segOf('def', 3)] // 拼接 "abcdef"

  it('起点落在节点内', () => {
    expect(locateOffset(segs, 2, false)).toEqual({ node: segs[0].node, offset: 2 })
  })

  it('起点落在边界归右侧节点开头', () => {
    expect(locateOffset(segs, 3, false)).toEqual({ node: segs[1].node, offset: 0 })
  })

  it('终点落在边界归左侧节点末尾', () => {
    expect(locateOffset(segs, 3, true)).toEqual({ node: segs[0].node, offset: 3 })
  })

  it('空段返回 null', () => {
    expect(locateOffset([], 0, false)).toBeNull()
  })
})

describe('buildTextIndex', () => {
  it('拼接文本并跳过 .katex-mathml', () => {
    const body = document.createElement('div')
    body.innerHTML = '<p>foo</p><span class="katex-mathml">HIDDEN</span><p>bar</p>'
    const idx = buildTextIndex(body)
    expect(idx.full).toBe('foobar')
    expect(idx.segs.length).toBe(2)
    expect(idx.segs[1].start).toBe(3)
  })

  it('bodyEl 为 null 返回空索引', () => {
    expect(buildTextIndex(null)).toEqual({ full: '', segs: [] })
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/__tests__/find.test.ts`
Expected: FAIL（无法解析 `../utils/markdown/find`）。

- [ ] **Step 3: 写实现**

创建 `frontend/src/utils/markdown/find.ts`：

```ts
// 从 vscode-office resource/markdown/find.js 移植纯逻辑，适配 TypeScript。

export interface MatchRange {
  start: number
  end: number
}
export interface TextSeg {
  node: Text
  start: number
}
export interface TextIndex {
  full: string
  segs: TextSeg[]
}

// 单次搜索匹配数上限：极大文档防卡死；达到时调用方应提示已截断。
export const MAX_MATCHES = 5000

function clamp(v: number, lo: number, hi: number): number {
  return v < lo ? lo : v > hi ? hi : v
}

/**
 * 在长文本中找出 query 的全部不重叠匹配，返回全局偏移区间 [{start,end}]（end 不含）。
 * 大小写不敏感时优先走快路径：整串 toLowerCase 后长度不变（绝大多数文本），偏移 1:1 对应。
 * 极少数 Unicode 折叠会改变长度（如 'İ'），走慢路径：逐字符折叠 + 反查映射回原串偏移。
 */
export function findMatches(full: string, query: string, caseSensitive: boolean): MatchRange[] {
  const out: MatchRange[] = []
  if (!full || !query) return out

  if (caseSensitive) {
    const step = query.length
    let from = 0
    let at: number
    while ((at = full.indexOf(query, from)) !== -1) {
      out.push({ start: at, end: at + step })
      from = at + step
      if (out.length >= MAX_MATCHES) break
    }
    return out
  }

  const needle = query.toLowerCase()
  const step = needle.length
  if (!step) return out

  const lowFull = full.toLowerCase()
  if (lowFull.length === full.length) {
    // 快路径
    let from = 0
    let at: number
    while ((at = lowFull.indexOf(needle, from)) !== -1) {
      out.push({ start: at, end: at + step })
      from = at + step
      if (out.length >= MAX_MATCHES) break
    }
    return out
  }

  // 慢路径：逐字符折叠 + 反查映射（back[k] = 折叠串第 k 个字符来自原串的下标）。
  let folded = ''
  const back: number[] = []
  for (let i = 0; i < full.length; i++) {
    const lc = full[i].toLowerCase()
    for (let k = 0; k < lc.length; k++) back.push(i)
    folded += lc
  }
  let from = 0
  let at: number
  while ((at = folded.indexOf(needle, from)) !== -1) {
    const start = back[at]
    const end = at + needle.length < back.length ? back[at + needle.length] : full.length
    out.push({ start, end })
    from = at + needle.length
    if (out.length >= MAX_MATCHES) break
  }
  return out
}

/**
 * 把全局偏移 pos 映射回某文本节点内的 (node, offset)。
 * atEnd 控制边界归属：起点(false)落边界归右侧节点开头；终点(true)落边界归左侧节点末尾，
 * 避免区间末端跨进无关后续块导致 getBoundingClientRect 把多块矩形并起来。
 */
export function locateOffset(
  segs: TextSeg[],
  pos: number,
  atEnd: boolean,
): { node: Text; offset: number } | null {
  if (!segs || !segs.length) return null
  for (let i = 0; i < segs.length; i++) {
    const seg = segs[i]
    const len = seg.node.nodeValue!.length
    const segEnd = seg.start + len
    const hit = atEnd ? pos <= segEnd : pos < segEnd
    if (hit || i === segs.length - 1) {
      return { node: seg.node, offset: clamp(pos - seg.start, 0, len) }
    }
  }
  const last = segs[segs.length - 1]
  return { node: last.node, offset: last.node.nodeValue!.length }
}

/**
 * 遍历 .md-body 内文本节点 → 拼长串 + 段映射，支持跨内联标签匹配。
 * 跳过 KaTeX 隐藏的 MathML 源码副本（.katex-mathml），避免看不见的重复匹配。
 */
export function buildTextIndex(bodyEl: HTMLElement | null): TextIndex {
  if (!bodyEl) return { full: '', segs: [] }
  const walker = document.createTreeWalker(bodyEl, NodeFilter.SHOW_TEXT, {
    acceptNode(node: Node): number {
      if (!node.nodeValue) return NodeFilter.FILTER_REJECT
      const pe = (node as Text).parentElement
      if (pe && pe.closest('.katex-mathml')) return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    },
  })
  let full = ''
  const segs: TextSeg[] = []
  let n: Node | null
  while ((n = walker.nextNode())) {
    const t = n as Text
    segs.push({ node: t, start: full.length })
    full += t.nodeValue
  }
  return { full, segs }
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd frontend && npx vitest run src/__tests__/find.test.ts`
Expected: PASS（全绿）。

- [ ] **Step 5: 提交**

```bash
git add frontend/src/utils/markdown/find.ts frontend/src/__tests__/find.test.ts
git commit -m "feat: 查找纯逻辑 findMatches/locateOffset/buildTextIndex + 单测

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 大纲面板组件 `MarkdownOutline.vue`

**说明：** 该组件依赖 `IntersectionObserver` 等浏览器 API，jsdom 无法可靠单测且本项目不引入 `@vue/test-utils`；本任务以 `npm run build`（vue-tsc 类型校验）为门禁，行为靠 Task 6 手动 QA 清单验证。

**Files:**
- Create: `frontend/src/components/MarkdownOutline.vue`

**Interfaces:**
- Consumes: `extractHeadings`, `buildOutlineTree`, `OutlineNode`（Task 1）。
- Produces（供 Task 5 宿主使用）:
  - props: `{ bodyEl: HTMLElement | null; scrollEl: HTMLElement | null; html: string; open: boolean; mode: 'push' | 'overlay'; width: number }`
  - emits: `update:open(boolean)`、`update:mode('push'|'overlay')`、`update:width(number)`、`available(boolean)`

- [ ] **Step 1: 写组件**

创建 `frontend/src/components/MarkdownOutline.vue`：

```vue
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
```

- [ ] **Step 2: 类型校验**

Run: `cd frontend && npm run build`
Expected: 通过（无 TS 报错；产物生成）。若组件未被任何文件 import 也不影响——本任务只验证编译，挂载在 Task 5。

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/MarkdownOutline.vue
git commit -m "feat: 大纲面板组件 MarkdownOutline.vue（折叠树/滚动联动/调宽/模式切换）

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: 查找条组件 `MarkdownFind.vue` + 全局高亮样式

**说明：** 依赖 CSS Custom Highlight API，jsdom 不支持；以 `npm run build` 类型校验为门禁，行为靠 Task 6 手动 QA。

**Files:**
- Create: `frontend/src/components/MarkdownFind.vue`
- Create: `frontend/src/assets/markdown/find-highlight.css`

**Interfaces:**
- Consumes: `findMatches`, `locateOffset`, `buildTextIndex`, `MAX_MATCHES`（Task 2）。
- Produces（供 Task 5 宿主使用）:
  - props: `{ bodyEl: HTMLElement | null; scrollEl: HTMLElement | null; open: boolean }`
  - emits: `update:open(boolean)`
  - expose: `openWith(prefill?: string): void`、`go(dir: number): void`

- [ ] **Step 1: 写全局高亮样式**

创建 `frontend/src/assets/markdown/find-highlight.css`：

```css
/* 全局（非 scoped）：::highlight 伪元素无法被 scoped CSS 命中，必须全局引入。
   底色高不透明度黄 + 强制深色文字，亮/暗主题下都清晰；当前项用更醒目的橙。 */
::highlight(md-find) {
  background: var(--md-find-bg, rgba(255, 210, 0, 0.55));
  color: #1a1a1a;
}
::highlight(md-find-current) {
  background: var(--md-find-current-bg, rgba(255, 145, 0, 0.9));
  color: #1a1a1a;
}
```

- [ ] **Step 2: 写组件**

创建 `frontend/src/components/MarkdownFind.vue`：

```vue
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

function onInput() {
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

// 晚到内容（图片/KaTeX/Mermaid 异步渲染）稳定后重跑一次（不滚动，避免跳动）。
function onWindowLoad() {
  if (props.open && query.value) runSearch(false)
}
onMounted(() => window.addEventListener('load', onWindowLoad))
onBeforeUnmount(() => {
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
```

- [ ] **Step 3: 类型校验**

Run: `cd frontend && npm run build`
Expected: 通过（无 TS 报错）。

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/MarkdownFind.vue frontend/src/assets/markdown/find-highlight.css
git commit -m "feat: 查找条组件 MarkdownFind.vue + 全局高亮样式（CSS Custom Highlight API）

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: 接入宿主 `MarkdownPreview.vue`

**Files:**
- Modify: `frontend/src/components/MarkdownPreview.vue`

**Interfaces:**
- Consumes: `MarkdownOutline`（Task 3 props/emits）、`MarkdownFind`（Task 4 props/emits/expose）、`find-highlight.css`（Task 4）。
- Produces: 暴露新增 `scrollEl`（既有 `defineExpose` 追加）；其余为内部接线。

> 下列 Edit 按文件现状（共 273 行）给出精确 old→new。逐个应用。

- [ ] **Step 1: 扩展 vue 与组件 import**

把第 2 行：

```ts
import { ref, watch, computed, onMounted, nextTick } from 'vue'
```

改为：

```ts
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
```

并在第 12 行 `import html2canvas from 'html2canvas'` 之后新增三行：

```ts
import MarkdownOutline from './MarkdownOutline.vue'
import MarkdownFind from './MarkdownFind.vue'
import '../assets/markdown/find-highlight.css'
```

- [ ] **Step 2: 新增 scrollEl 与大纲/查找状态**

在第 22 行 `const rootEl = ref<HTMLElement | null>(null)` 之后插入：

```ts
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
```

- [ ] **Step 3: 注册/注销全局快捷键监听**

把第 95–98 行：

```ts
onMounted(async () => {
  await nextTick()
  await runMermaid()
})
```

改为：

```ts
onMounted(async () => {
  await nextTick()
  await runMermaid()
  window.addEventListener('keydown', onGlobalKeydown, true)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeydown, true)
})
```

- [ ] **Step 4: defineExpose 追加 scrollEl**

把第 187 行：

```ts
defineExpose({ rootEl, bodyEl, theme })
```

改为：

```ts
defineExpose({ rootEl, bodyEl, scrollEl, theme })
```

- [ ] **Step 5: 模板——预览根挂 hover、滚动容器加 ref + 推送、挂载两组件**

把模板根块（第 191–195 行）：

```html
  <div class="md-preview-root" :data-theme="theme" ref="rootEl">
    <div class="md-scroll" @wheel="onWheel">
      <div class="md-body" ref="bodyEl" :style="{ fontSize: bodyFontPx + 'px' }"
           v-html="html" @click="onClick"></div>
    </div>
```

改为：

```html
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
```

- [ ] **Step 6: 工具栏新增 📑 大纲 / 🔍 查找按钮**

把工具栏块（第 198–204 行）：

```html
    <!-- 工具条（左→右）：主题 / 导出 / 缩小 / 放大 / 刷新 -->
    <div class="md-toolbar">
      <button class="md-tool-btn" title="主题" @click="toggleTheme">🎨</button>
      <button class="md-tool-btn" title="导出" @click="toggleExport">📤</button>
      <button class="md-tool-btn" title="缩小" @click="zoomOut">➖</button>
      <button class="md-tool-btn" title="放大" @click="zoomIn">➕</button>
      <button class="md-tool-btn" title="刷新" @click="doRefresh">🔄</button>
    </div>
```

改为：

```html
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
```

- [ ] **Step 7: 样式——滚动容器加推送过渡 + 工具按钮禁用态**

把 `.md-scroll` 样式块（第 235–238 行）：

```css
.md-scroll {
  height: 100%;
  overflow: auto;
}
```

改为：

```css
.md-scroll {
  height: 100%;
  overflow: auto;
  transition: padding-left 0.2s ease;
}
```

并在 `.md-tool-btn:hover { opacity: 1; }`（第 248 行）之后新增一行：

```css
.md-tool-btn.disabled { opacity: 0.4; pointer-events: none; }
```

- [ ] **Step 8: 类型校验**

Run: `cd frontend && npm run build`
Expected: 通过（无 TS 报错；vite 产物生成）。

- [ ] **Step 9: 提交**

```bash
git add frontend/src/components/MarkdownPreview.vue
git commit -m "feat: 预览接入大纲/查找（工具栏 📑🔍、Ctrl+F、推送布局、状态持久化）

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: 全量校验 + 手动 QA + 文档

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: 全量单测 + 类型/构建校验**

Run: `cd frontend && npm run test && npm run build`
Expected: 测试全绿（含 `render` / `outline` / `find` 三套）；构建通过。

- [ ] **Step 2: 手动 QA（`wails dev`，逐项核对）**

启动：`wails dev`（仓库根）。打开一个含多级标题、代码块、KaTeX、Mermaid、长文本的 `.md` 文件，逐项验证：

大纲：
- [ ] 工具栏 📑 切换面板；无标题文档时 📑 置灰、面板与把手不出现。
- [ ] 大纲关闭时左缘出现 ☰，点击展开。
- [ ] 点标题平滑滚动到对应位置；滚动正文时大纲活跃项跟随高亮。
- [ ] 折叠/展开 caret 生效；滚到被折叠的子标题时其祖先自动展开。
- [ ] 拖右缘 grip 调宽（180–480 间），松手后刷新预览宽度仍保持（localStorage）。
- [ ] ⇆ 切换推送/浮层：推送模式正文右移不被遮挡；浮层模式面板浮于正文上。
- [ ] 预览「上下/左右」两种布局、窄面板下，面板与把手均锚定在预览内、不溢出到终端。

查找：
- [ ] 鼠标在预览内按 Ctrl+F 打开查找条；焦点在终端时按 Ctrl+F 不触发（终端不受影响）。
- [ ] 工具栏 🔍 也能打开。
- [ ] 选中一段文字后 Ctrl+F 预填该文字并立即出结果。
- [ ] 输入即高亮全部匹配 + 当前项更醒目；计数 `N / M` 正确；无结果显示「无结果」。
- [ ] Enter / ↓ 下一个、Shift+Enter / ↑ 上一个、F3 / Shift+F3 跳转，均环绕；当前项滚动居中。
- [ ] Aa 区分大小写切换即时重搜。
- [ ] Esc / ✕ 关闭并清除高亮；关闭后键盘仍可滚动正文。
- [ ] 跨内联标签（如含 `code`/加粗的句子）能整体命中；公式不出现重复匹配。
- [ ] 切换多套亮/暗主题，高亮对比度均清晰。

- [ ] **Step 3: 更新 CHANGELOG**

在 `CHANGELOG.md` 顶部新增一条（紧接标题/说明之后、现有最新版本之前）：

```markdown
## [Unreleased]

### Added
- Markdown 预览新增**大纲目录**：左侧可折叠标题树，滚动联动高亮，推送/浮层模式，拖拽调宽，左缘把手 + 工具栏 📑；开关/模式/宽度记忆到 localStorage。
- Markdown 预览新增 **Ctrl+F 查找**：预览内查找条，跨节点匹配、命中计数、区分大小写、↑/↓/F3 导航、Esc 关闭，CSS Custom Highlight API 着色；工具栏 🔍 入口。
```

> 注：若 `CHANGELOG.md` 既有结构与上述标题层级不一致，按其既有写法对齐（保持同级标题与中文风格），内容不变。

- [ ] **Step 4: 提交**

```bash
git add CHANGELOG.md
git commit -m "docs: CHANGELOG 记录 Markdown 预览大纲目录 + Ctrl+F 查找

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review（计划编写者自查，已完成）

**1. Spec coverage：**
- 大纲：建树/扫描（T1）、面板/折叠/联动/调宽/模式/把手（T3）、宿主接线/工具栏/推送/持久化（T5）✓
- 查找：匹配/定位/索引（T2）、查找条/高亮/导航/计数（T4）、Ctrl+F 接线/工具栏/终端隔离（T5）✓
- 5 项关键适配：fixed→absolute（T3/T4 样式）、滚动监听 .md-scroll（T3）、推送作用 .md-scroll（T5）、Ctrl+F 隔离（T5）、移除降级路径（T4 仅 Highlight API）✓
- 持久化键、非目标（导出不变/无新依赖/查找不持久化）、测试计划 ✓

**2. Placeholder scan：** 无 TBD/TODO；每个代码步骤给出完整代码；CHANGELOG 步骤给出确切文案。✓

**3. Type consistency：**
- `OutlineItem`/`OutlineNode`（T1）↔ `MarkdownOutline` 消费（T3）一致。
- `MatchRange`/`TextSeg`/`TextIndex`/`MAX_MATCHES`/`findMatches`/`locateOffset`/`buildTextIndex`（T2）↔ `MarkdownFind` 消费（T4）一致。
- 组件 props/emits/expose（`openWith`/`go`/`available`/`update:open|mode|width`）↔ 宿主接线（T5）一致。
- localStorage 键三处统一 `md-preview-outline-open|mode|width`。✓
