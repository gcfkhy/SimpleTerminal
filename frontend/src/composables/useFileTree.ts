import { ref, reactive } from 'vue'
import { ReadDir, OpenFolderDialog } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

// 树节点：在后端 FileEntry（name/path/isDir）上扩展出树形所需的状态。
export interface TreeNode {
  name: string
  path: string
  isDir: boolean
  children: TreeNode[] | null // null = 尚未加载（懒加载标记）
  expanded: boolean
  loading: boolean
}

const ROOT_KEY = 'fileTreeRoot'
const EXPANDED_KEY = 'fileTreeExpanded'

// 模块级单例：App.vue 与 FileTree/FileTreeNode 共享同一份状态（故意定义在 useFileTree 外）。
const rootNode = ref<TreeNode | null>(null)
const currentPath = ref('') // 当前树根路径（App.vue 的 cd gate 与「在当前目录新建标签」依赖）
const selectedFile = ref<main.FileEntry | null>(null)
const selectedPath = ref('')
const error = ref('')
const pickedDir = ref('')
const expandedPaths = new Set<string>()

// 去掉末尾分隔符，返回 [主体, 最后一个 \ 或 / 的下标]（找不到为 -1）。
function splitPath(p: string): readonly [string, number] {
  const body = p.replace(/[\\/]+$/, '')
  return [body, Math.max(body.lastIndexOf('\\'), body.lastIndexOf('/'))]
}

function baseName(p: string): string {
  const [body, i] = splitPath(p)
  return i >= 0 ? body.slice(i + 1) : body
}

function parentPath(p: string): string {
  const [body, i] = splitPath(p)
  if (i <= 0) return ''
  const parent = body.slice(0, i)
  return /^[a-zA-Z]:$/.test(parent) ? parent + '\\' : parent
}

function toNode(e: main.FileEntry): TreeNode {
  return { name: e.name, path: e.path, isDir: e.isDir, children: null, expanded: false, loading: false }
}

function persistExpanded() {
  localStorage.setItem(EXPANDED_KEY, JSON.stringify([...expandedPaths]))
}
function loadExpandedFromStorage() {
  try {
    const arr = JSON.parse(localStorage.getItem(EXPANDED_KEY) || '[]')
    if (Array.isArray(arr)) {
      expandedPaths.clear()
      for (const p of arr) if (typeof p === 'string') expandedPaths.add(p)
    }
  } catch {
    /* 忽略损坏的持久化数据 */
  }
}

export function useFileTree() {
  // 懒加载某节点的子项；失败时置空 children 并记录错误。
  async function loadChildren(node: TreeNode): Promise<boolean> {
    node.loading = true
    try {
      const list = await ReadDir(node.path)
      node.children = (list ?? []).map(toNode)
      error.value = ''
      return true
    } catch (e) {
      error.value = String(e)
      node.children = []
      return false
    } finally {
      node.loading = false
    }
  }

  // 启动恢复：对记住的、位于当前子树下的展开路径，逐层加载并展开。
  async function restoreExpanded(node: TreeNode): Promise<void> {
    if (!node.children) return
    for (const child of node.children) {
      if (child.isDir && expandedPaths.has(child.path)) {
        child.expanded = true
        await loadChildren(child)
        await restoreExpanded(child)
      }
    }
  }

  // 把树根设为 path 并加载（旧名 loadDir：终端 cd 同步、选目录都走这里）。
  async function loadDir(path: string): Promise<boolean> {
    currentPath.value = path // 同步设根，让 App.vue 的 cd gate 立即关闭，避免竞态
    // 必须用 reactive 包装：loadChildren 是直接改 root.children，
    // 只有响应式代理才会触发界面更新，否则懒加载的子节点出不来、树空白（踩过的坑）。
    const root = reactive<TreeNode>({
      name: baseName(path) || path,
      path,
      isDir: true,
      children: null,
      expanded: true,
      loading: false,
    })
    rootNode.value = root
    localStorage.setItem(ROOT_KEY, path)
    const ok = await loadChildren(root)
    if (ok) await restoreExpanded(root)
    return ok
  }

  // 展开/折叠目录；首次展开才懒加载子项。
  async function toggleDir(node: TreeNode): Promise<void> {
    if (!node.isDir) return
    node.expanded = !node.expanded
    if (node.expanded) {
      expandedPaths.add(node.path)
      if (node.children === null) await loadChildren(node)
    } else {
      expandedPaths.delete(node.path)
    }
    persistExpanded()
  }

  // 点击一行：目录 → 展开/折叠；文件 → 切换预览（TreeNode 结构兼容 FileEntry）。
  function open(node: TreeNode): void {
    selectedPath.value = node.path
    if (node.isDir) {
      void toggleDir(node)
    } else {
      selectedFile.value = selectedFile.value?.path === node.path ? null : node
    }
  }

  // 把树根换成上一级目录。
  function goUp(): void {
    const parent = parentPath(currentPath.value)
    if (parent && parent !== currentPath.value) void loadDir(parent)
  }

  async function openFolderDialog(): Promise<void> {
    const picked = await OpenFolderDialog()
    if (picked) {
      await loadDir(picked)
      pickedDir.value = picked
    }
  }

  // 启动初始化：恢复上次根目录与展开状态。
  function init(): void {
    loadExpandedFromStorage()
    const savedRoot = localStorage.getItem(ROOT_KEY)
    if (savedRoot) void loadDir(savedRoot)
  }

  return {
    rootNode,
    currentPath,
    error,
    selectedFile,
    selectedPath,
    pickedDir,
    init,
    loadDir,
    toggleDir,
    open,
    goUp,
    openFolderDialog,
  }
}