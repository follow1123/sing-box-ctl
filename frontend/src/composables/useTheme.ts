import { ref } from 'vue'

const THEME_KEY = 'sbctl-theme'

export type Theme = 'auto' | 'light' | 'dark'

/** 当前主题选择（响应式，组件可 watch） */
export const theme = ref<Theme>((localStorage.getItem(THEME_KEY) as Theme) || 'auto')

const mq = window.matchMedia('(prefers-color-scheme: dark)')

/** 解析当前应实际应用的亮/暗（auto 跟随系统） */
export function isDark(): boolean {
  return theme.value === 'dark' || (theme.value === 'auto' && mq.matches)
}

/** 将解析结果写入 <html data-theme="light|dark">，驱动 tokens.css 变量 */
export function applyTheme(): void {
  document.documentElement.dataset.theme = isDark() ? 'dark' : 'light'
}

/** 切换主题并持久化 */
export function setTheme(t: Theme): void {
  theme.value = t
  localStorage.setItem(THEME_KEY, t)
  applyTheme()
}

// auto 模式下跟随系统切换
mq.addEventListener('change', () => {
  if (theme.value === 'auto') applyTheme()
})
