import { ref } from 'vue'

const THEME_KEY = 'sbctl-theme'

export type Theme = 'auto' | 'light' | 'dark'

/** 当前主题选择（响应式，组件可 watch） */
export const theme = ref<Theme>((localStorage.getItem(THEME_KEY) as Theme) || 'auto')

/** 当前是否实际处于暗色（自动模式跟随系统） */
export function isDark(): boolean {
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  return theme.value === 'dark' || (theme.value === 'auto' && mq.matches)
}

/** 将主题应用到 <html data-theme>，驱动全局 CSS 变量 */
export function applyTheme(): void {
  document.documentElement.dataset.theme = theme.value === 'auto' ? '' : theme.value
}

/** 切换主题并持久化 */
export function setTheme(t: Theme): void {
  theme.value = t
  localStorage.setItem(THEME_KEY, t)
  applyTheme()
}
