import { ref } from 'vue'
import { ReadDir, OpenFolderDialog } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

// 树节点：在后端 FileEntry（name/path/isDir）基础上扩展出树形所需的状态。
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

// 模块级单例：App.vue 与 FileTree/FileTreeNode 共享同一份状态。
const rootNode = ref<TreeNode | null>(null)
const currentPath = ref<string>('')
const selectedFile = ref<main.FileEntry | null>(null)
const selectedPath = ref<string>('')
const error = ref<string>('')
const pickedDir = ref('')

const expandedPaths = new Set<string>()

function baseName(p: string): string {
  const n = p.replace(/[\\/]+$/, '')
  const i = Math.max(n.lastIndexOf('\\'), n.lastIndexOf('/'))
  return i >= 0 ? n.slice(i + 1) : n
}

function parentPath(p: string): string {
  const n = p.replace(/[\\/]+$/, '')
  const i = Math.max(n.lastIndexOf('\\'), n.lastIndexOf('/'))
  if (i <= 0) return ''
  let parent = n.slice(0, i)
  if (/^[a-zA-Z]:$/.test(parent)) parent += '\\'
  return parent
}

function toNode(e: main.FileEntry): TreeNode {
  return { name: e.name, path: e.path, isDir: e.isDir, children: null, expanded: false, loading: false }
}

function persistRoot(path: string) {
  localStorage.setItem(ROOT_KEY, path)
}
function persistExpanded() {
  localStorage.setItem(EXPANDED_KEY, JSON.stringify(Array.from(expandedPaths)))
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

  // 把树根设为 path 并加载。沿用旧名 loadDir：cd 同步、选目录都走这里。
  async function loadDir(path: string): Promise<boolean> {
    currentPath.value = path
    const root: TreeNode = {
      name: baseName(path) || path,
      path,
      isDir: true,
      children: null,
      expanded: true,
      loading: false,
    }
    rootNode.value = root
    persistRoot(path)
    const ok = await loadChildren(root)
    if (ok) await restoreExpanded(root)
    return ok
  }

  // 展开/折叠目录；首次展开时懒加载子项。
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

  // 点击一行：目录 → 展开/折叠；文件 → 切换预览。
  function open(node: TreeNode): void {
    selectedPath.value = node.path
    if (node.isDir) {
      void toggleDir(node)
    } else {
      selectedFile.value = selectedFile.value?.path === node.path ? null : node
    }
  }

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
</parameter>
</invoke>
