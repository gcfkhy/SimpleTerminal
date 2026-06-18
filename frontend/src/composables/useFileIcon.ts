// 把文件名/扩展名映射到 Material Icon Theme 的 SVG（jsDelivr CDN 加载）。
// <img @error> 会回退到通用文件/文件夹图标。
const CDN = 'https://cdn.jsdelivr.net/gh/PKief/vscode-material-icon-theme/icons'

const FILE_FALLBACK = `${CDN}/file.svg`
const FOLDER_FALLBACK = `${CDN}/folder.svg`

// 扩展名 → 图标名
const extMap: Record<string, string> = {
  ts: 'typescript', tsx: 'react_ts', js: 'javascript', jsx: 'react', mjs: 'javascript',
  vue: 'vue', json: 'json', md: 'markdown', html: 'html', htm: 'html',
  css: 'css', scss: 'sass', sass: 'sass', less: 'less',
  go: 'go', py: 'python', rs: 'rust', java: 'java', kt: 'kotlin',
  c: 'c', h: 'h', cpp: 'cpp', cc: 'cpp', hpp: 'hpp', cs: 'csharp',
  php: 'php', rb: 'ruby', swift: 'swift', dart: 'dart', lua: 'lua',
  sh: 'console', bash: 'console', zsh: 'console', ps1: 'powershell', bat: 'console', cmd: 'console',
  yml: 'yaml', yaml: 'yaml', toml: 'toml', ini: 'settings', xml: 'xml', conf: 'settings',
  sql: 'database', db: 'database', csv: 'table', tsv: 'table',
  png: 'image', jpg: 'image', jpeg: 'image', gif: 'image', webp: 'image', bmp: 'image', ico: 'image', svg: 'svg',
  pdf: 'pdf', doc: 'word', docx: 'word', xls: 'table', xlsx: 'table', ppt: 'powerpoint', pptx: 'powerpoint',
  zip: 'zip', rar: 'zip', '7z': 'zip', tar: 'zip', gz: 'zip',
  mp4: 'video', mkv: 'video', avi: 'video', mov: 'video', webm: 'video', flv: 'video',
  mp3: 'audio', wav: 'audio', flac: 'audio', ogg: 'audio',
  txt: 'document', log: 'log', exe: 'exe', dll: 'dll', lock: 'lock', env: 'tune',
}

// 完整文件名（小写）→ 图标名
const nameMap: Record<string, string> = {
  'package.json': 'nodejs',
  'package-lock.json': 'nodejs',
  'tsconfig.json': 'tsconfig',
  'tsconfig.node.json': 'tsconfig',
  'vite.config.ts': 'vite',
  'vite.config.js': 'vite',
  '.gitignore': 'git',
  '.gitattributes': 'git',
  'go.mod': 'go-mod',
  'go.sum': 'go-mod',
  'readme.md': 'readme',
  'license': 'certificate',
  'license.txt': 'certificate',
  'dockerfile': 'docker',
  '.env': 'tune',
  'wails.json': 'json',
}

export function useFileIcon() {
  function getIcon(name: string, isDir: boolean): string {
    if (isDir) return FOLDER_FALLBACK
    const lower = name.toLowerCase()
    if (nameMap[lower]) return `${CDN}/${nameMap[lower]}.svg`
    const dot = lower.lastIndexOf('.')
    const ext = dot >= 0 ? lower.slice(dot + 1) : ''
    if (ext && extMap[ext]) return `${CDN}/${extMap[ext]}.svg`
    return FILE_FALLBACK
  }

  function fallback(isDir: boolean): string {
    return isDir ? FOLDER_FALLBACK : FILE_FALLBACK
  }

  return { getIcon, fallback }
}
