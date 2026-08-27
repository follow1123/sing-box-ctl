<script setup lang="ts">
import { theme, setTheme } from '../composables/useTheme'
</script>

<template>
  <select
    class="theme-switch"
    :value="theme"
    title="主题"
    @change="setTheme(($event.target as HTMLSelectElement).value as typeof theme)"
  >
    <option value="auto">自动</option>
    <option value="light">亮色</option>
    <option value="dark">暗色</option>
  </select>
</template>

<style scoped>
.theme-switch {
  margin-left: auto;
  padding: 0.3rem 0.5rem;
  border: 1px solid var(--surface-3);
  border-radius: var(--radius-2);
  background: var(--surface-2);
  color: var(--text-1);
  font-size: 0.85rem;
  cursor: pointer;
}
</style>
