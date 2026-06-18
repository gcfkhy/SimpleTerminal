import { ref } from 'vue'
import { ReadDir, OpenFolderDialog } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

const pickedDir   = ref('')
const selectedFile = ref<main.FileEntry | null>(null)

export function useFileTree() {
  const currentPath = ref<string>('')
  const entries = ref<main.FileEntry[]>([])
  const error = ref<string>('')

  async function loadDir(path: string): Promise<boolean> {
    try {
      const list = await ReadDir(path)
      entries.value = list ?? []
      currentPath.value = path
      error.value = ''
      return true
    } catch (e) {
      error.value = String(e)
      return false
    }
  }

  function init() {
    // 启动时不加载任何目录，等待用户点击"选择目录"按钮
  }

  function goUp() {
    const p = currentPath.value
    if (!p) return
    const normalized = p.replace(/[\\/]+$/, '')
    const idx = Math.max(normalized.lastIndexOf('\\'), normalized.lastIndexOf('/'))
    if (idx <= 0) return
    let parent = normalized.slice(0, idx)
    if (/^[a-zA-Z]:$/.test(parent)) parent += '\\'
    void loadDir(parent)
  }

  async function openFolderDialog() {
    const picked = await OpenFolderDialog()
    if (picked) {
      await loadDir(picked)
      pickedDir.value = picked  // 通知 App.vue 向活跃 Tab 发送 cd
    }
  }

  function open(entry: main.FileEntry) {
    if (entry.isDir) {
      void loadDir(entry.path)
    } else {
      // 点击文件：切换预览（再次点击同一文件则关闭）
      selectedFile.value = selectedFile.value?.path === entry.path ? null : entry
    }
  }

  return { currentPath, entries, error, init, loadDir, goUp, openFolderDialog, open, pickedDir, selectedFile }
}
