import { ref } from 'vue'
import { ReadDir, OpenFolderDialog, HomeDir } from '../../wailsjs/go/main/App'
import { EventsEmit } from '../../wailsjs/runtime'
import type { main } from '../../wailsjs/go/models'

const LAST_DIR_KEY = 'lastDir'

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

  // 挂载时恢复上次目录，失败则回退到用户主目录
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

  // 上级目录（兼容 Windows 反斜杠/正斜杠，盘符根停止）
  function goUp() {
    const p = currentPath.value
    if (!p) return
    const normalized = p.replace(/[\\/]+$/, '')
    const idx = Math.max(normalized.lastIndexOf('\\'), normalized.lastIndexOf('/'))
    if (idx <= 0) return
    let parent = normalized.slice(0, idx)
    if (/^[a-zA-Z]:$/.test(parent)) parent += '\\' // "C:" → "C:\"
    void loadDir(parent)
  }

  async function openFolderDialog() {
    const picked = await OpenFolderDialog()
    if (picked) await loadDir(picked)
  }

  // 点击：目录则进入并在终端 cd；文件不做处理（v2.0 预览面板再用）
  function open(entry: main.FileEntry) {
    if (entry.isDir) {
      EventsEmit('pty:input', `cd "${entry.path}"\r`)
      void loadDir(entry.path)
    }
  }

  return { currentPath, entries, error, init, loadDir, goUp, openFolderDialog, open }
}
