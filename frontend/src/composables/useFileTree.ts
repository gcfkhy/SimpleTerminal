import { ref } from 'vue'
import { ReadDir, OpenFolderDialog, HomeDir } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'

const LAST_DIR_KEY = 'lastDir'

// 模块级共享：App.vue 监听此 ref，在当前活跃 Tab 执行 cd
const pickedDir = ref('')

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
      localStorage.setItem(LAST_DIR_KEY, path)
      return true
    } catch (e) {
      error.value = String(e)
      return false
    }
  }

  async function init() {
    const last = localStorage.getItem(LAST_DIR_KEY)
    if (last && (await loadDir(last))) return
    try {
      const home = await HomeDir()
      await loadDir(home)
    } catch (e) {
      error.value = String(e)
    }
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
    }
  }

  return { currentPath, entries, error, init, loadDir, goUp, openFolderDialog, open, pickedDir }
}
