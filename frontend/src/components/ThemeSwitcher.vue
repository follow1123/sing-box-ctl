<script setup lang="ts">
import { theme, setTheme } from '../composables/useTheme'
import Btn from './ui/Btn.vue'

const order = ['auto', 'light', 'dark'] as const

function cycle(): void {
  const i = order.indexOf(theme.value)
  setTheme(order[(i + 1) % order.length])
}
</script>

<template>
  <Btn
    variant="ghost"
    size="sm"
    square
    title="切换主题（自动 / 亮色 / 暗色）"
    @click="cycle"
  >
    <!-- 自动：左浅右深的明暗分半圆（不依赖背景色，悬停/主题下都自然） -->
    <svg v-if="theme === 'auto'" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2a10 10 0 1 1 0 20A10 10 0 0 1 12 2z" fill-opacity="0.22" />
      <path d="M12 2a10 10 0 0 1 10 10H12z" />
    </svg>
    <!-- 亮色：太阳 -->
    <svg
      v-else-if="theme === 'light'"
      class="ti"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="4" />
      <path
        d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"
      />
    </svg>
    <!-- 暗色：月亮 -->
    <svg
      v-else
      class="ti"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  </Btn>
</template>

<style scoped>
.ti {
  width: 1.1rem;
  height: 1.1rem;
}
</style>
