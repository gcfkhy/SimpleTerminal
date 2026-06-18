const BASE = 'https://cdn.jsdelivr.net/gh/material-extensions/vscode-material-icon-theme@main/icons'

const EXT: Record<string, string> = {
  // Scripts / Languages
  js: 'javascript', mjs: 'javascript', cjs: 'javascript',
  ts: 'typescript', mts: 'typescript',
  tsx: 'react_ts', jsx: 'react',
  vue: 'vue', svelte: 'svelte',
  py: 'python', pyw: 'python',
  go: 'go',
  rs: 'rust',
  java: 'java', kt: 'kotlin', kts: 'kotlin',
  swift: 'swift',
  cpp: 'cpp', cc: 'cpp', cxx: 'cpp',
  c: 'c', h: 'h', hpp: 'hpp',
  cs: 'csharp',
  rb: 'ruby',
  php: 'php',
  sh: 'shell', bash: 'shell', zsh: 'shell', fish: 'shell',
  ps1: 'powershell', psm1: 'powershell',
  lua: 'lua',
  r: 'r',
  dart: 'dart',
  scala: 'scala',
  zig: 'zig',
  ex: 'elixir', exs: 'elixir',
  // Web
  html: 'html', htm: 'html',
  css: 'css',
  scss: 'scss', sass: 'sass', less: 'less',
  svg: 'svg',
  // Data / Config
  json: 'json', jsonc: 'json',
  yaml: 'yaml', yml: 'yaml',
  toml: 'toml',
  xml: 'xml',
  csv: 'table',
  sql: 'sql',
  // Docs
  md: 'markdown', mdx: 'mdx',
  txt: 'text',
  pdf: 'pdf',
  // Images
  png: 'image', jpg: 'image', jpeg: 'image',
  gif: 'image', webp: 'image', ico: 'image', bmp: 'image',
  // Archives
  zip: 'zip', tar: 'zip', gz: 'zip', '7z': 'zip', rar: 'zip',
  // Misc
  exe: 'exe', dll: 'binary',
  wasm: 'wasm',
  lock: 'lock',
  log: 'log',
}

const FILENAME: Record<string, string> = {
  'package.json': 'nodejs',
  'package-lock.json': 'nodejs',
  'tsconfig.json': 'tsconfig',
  'tsconfig.node.json': 'tsconfig',
  '.env': 'env',
  '.env.local': 'env',
  '.gitignore': 'git',
  '.gitattributes': 'git',
  '.gitmodules': 'git',
  'dockerfile': 'docker',
  'docker-compose.yml': 'docker',
  'docker-compose.yaml': 'docker',
  'makefile': 'makefile',
  'cargo.toml': 'rust',
  'cargo.lock': 'rust',
  'go.mod': 'go',
  'go.sum': 'go',
  'requirements.txt': 'python',
  'pipfile': 'python',
  'readme.md': 'readme',
  'license': 'certificate',
  'license.md': 'certificate',
  'vite.config.ts': 'vite',
  'vite.config.js': 'vite',
  'wails.json': 'config',
}

const FOLDER: Record<string, string> = {
  'src': 'folder-src',
  'source': 'folder-src',
  'node_modules': 'folder-node',
  '.git': 'folder-git',
  '.github': 'folder-github',
  'dist': 'folder-dist',
  'build': 'folder-build',
  'out': 'folder-out',
  'public': 'folder-public',
  'assets': 'folder-resource',
  'components': 'folder-components',
  'views': 'folder-views',
  'pages': 'folder-views',
  'hooks': 'folder-hook',
  'composables': 'folder-hook',
  'utils': 'folder-utils',
  'helpers': 'folder-utils',
  'lib': 'folder-lib',
  'libs': 'folder-lib',
  'types': 'folder-typescript',
  'interfaces': 'folder-typescript',
  'tests': 'folder-test',
  'test': 'folder-test',
  '__tests__': 'folder-test',
  'scripts': 'folder-scripts',
  'styles': 'folder-styles',
  'css': 'folder-styles',
  'docs': 'folder-docs',
  'doc': 'folder-docs',
  'config': 'folder-config',
  'configs': 'folder-config',
  '.vscode': 'folder-vscode',
  '.claude': 'folder-vscode',
  'images': 'folder-images',
  'img': 'folder-images',
  'icons': 'folder-images',
  'wailsjs': 'folder-js',
}

export function getFileIcon(name: string, isDir: boolean): string {
  if (isDir) {
    const icon = FOLDER[name.toLowerCase()] ?? 'folder'
    return `${BASE}/${icon}.svg`
  }
  const lower = name.toLowerCase()
  if (FILENAME[lower]) return `${BASE}/${FILENAME[lower]}.svg`
  const ext = lower.includes('.') ? lower.split('.').pop()! : ''
  return `${BASE}/${EXT[ext] ?? 'file'}.svg`
}
