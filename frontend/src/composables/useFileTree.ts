import { ref, computed } from 'vue'
import { ReadDir } from '../../wailsjs/go/main/App'
import { EventsEmit } from '../../wailsjs/runtime'

export interface FileEntry {
  name: string
  path: string
  isDir: boolean
}

function getParentDir(p: string): string | null {
  if (!p) return null
  if (/^[A-Za-z]:\\?$/.test(p)) return null
  const lastSep = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'))
  if (lastSep <= 0) return null
  return p.substring(0, lastSep) || null
}

export function useFileTree() {
  const entries = ref<FileEntry[]>([])
  const currentPath = ref('')
  const parentDir = computed(() => getParentDir(currentPath.value))

  async function loadDir(path: string) {
    const result = await ReadDir(path)
    entries.value = result || []
    currentPath.value = path
    localStorage.setItem('lastDir', path)
  }

  function cdToDir(path: string) {
    EventsEmit('pty:input', `cd "${path}"\r`)
    loadDir(path)
  }

  return { entries, currentPath, parentDir, loadDir, cdToDir }
}
