import { ref } from 'vue'

export type ToastType = 'info' | 'warn' | 'error'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
}

const MAX = 3
const DURATION: Record<ToastType, number> = {
  info: 3000,
  warn: 5000,
  error: 8000,
}

/** 当前可见通知（ToastHost 消费） */
export const toasts = ref<ToastItem[]>([])

let seq = 0

export function dismiss(id: number): void {
  const i = toasts.value.findIndex((t) => t.id === id)
  if (i >= 0) toasts.value.splice(i, 1)
}

function push(type: ToastType, message: string): void {
  const item: ToastItem = { id: ++seq, type, message }
  toasts.value.push(item)
  // 最多同时 3 条：超出时移除最旧
  if (toasts.value.length > MAX) toasts.value.shift()
  setTimeout(() => dismiss(item.id), DURATION[type])
}

/** 编程式提示入口：toast.info / toast.warn / toast.error */
export const toast = {
  info: (message: string) => push('info', message),
  warn: (message: string) => push('warn', message),
  error: (message: string) => push('error', message),
}
