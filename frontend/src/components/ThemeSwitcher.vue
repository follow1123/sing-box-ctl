<script setup lang="ts">
import { theme, setTheme } from '../composables/useTheme'

const order = ['auto', 'light', 'dark'] as const

function cycle(): void {
  const i = order.indexOf(theme.value)
  setTheme(order[(i + 1) % order.length])
}
</script>

<template>
  <button class="theme-switch" title="切换主题（自动 / 亮色 / 暗色）" @click="cycle">
    <!-- 自动：半日半月的简化示意 -->
    <svg v-if="theme === 'auto'" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2a10 10 0 1 1 0 20V2z" />
    </svg>
    <!-- 亮色：太阳 -->
    <svg
      v-else-if="theme === 'light'"
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
  </button>
</template>

<style scoped>
.theme-switch {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.8rem;
  height: 1.8rem;
  margin-left: auto;
  padding: 0;
  background: none;
  border: none;
  border-radius: var(--radius-1);
  color: var(--text-2);
  cursor: pointer;
}
.theme-switch:hover {
  background: var(--surface-3);
  color: var(--text-1);
}
.theme-switch svg {
  width: 1.05rem;
  height: 1.05rem;
}
</style>
