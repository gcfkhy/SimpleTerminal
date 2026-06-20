export interface MarkdownTheme { id: string; name: string; group: 'light' | 'dark' }
export const DEFAULT_THEME_ID = 'catppuccin-mocha'
export const MARKDOWN_THEMES: MarkdownTheme[] = [
  { id: 'catppuccin-mocha', name: 'Catppuccin Mocha', group: 'dark' },
  { id: 'catppuccin-macchiato', name: 'Catppuccin Macchiato', group: 'dark' },
  { id: 'catppuccin-frappe', name: 'Catppuccin Frappé', group: 'dark' },
  { id: 'dracula', name: 'Dracula', group: 'dark' },
  { id: 'nord', name: 'Nord', group: 'dark' },
  { id: 'one-dark', name: 'One Dark', group: 'dark' },
  { id: 'tokyo-night', name: 'Tokyo Night', group: 'dark' },
  { id: 'gruvbox-dark', name: 'Gruvbox Dark', group: 'dark' },
  { id: 'solarized-dark', name: 'Solarized Dark', group: 'dark' },
  { id: 'rose-pine', name: 'Rosé Pine', group: 'dark' },
  { id: 'github-light', name: 'GitHub Light', group: 'light' },
  { id: 'catppuccin-latte', name: 'Catppuccin Latte', group: 'light' },
  { id: 'solarized-light', name: 'Solarized Light', group: 'light' },
  { id: 'gruvbox-light', name: 'Gruvbox Light', group: 'light' },
  { id: 'one-light', name: 'One Light', group: 'light' },
  { id: 'rose-pine-dawn', name: 'Rosé Pine Dawn', group: 'light' },
  { id: 'ayu-light', name: 'Ayu Light', group: 'light' },
  { id: 'tokyo-night-light', name: 'Tokyo Night Light', group: 'light' },
]
