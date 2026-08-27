<script setup lang="ts">
import ThemeSwitcher from './ThemeSwitcher.vue'

defineProps<{ active: string }>()
</script>

<template>
  <nav>
    <span class="brand">sbctl</span>
    <a href="#/" :class="{ active: active === '/' }">Provider 管理</a>
    <a href="#/templates" :class="{ active: active === '/templates' }">模板管理</a>
    <a href="#/url" :class="{ active: active === '/url' }">生成 URL</a>
    <ThemeSwitcher />
  </nav>
</template>

<style scoped>
nav {
  background: var(--surface-2);
  border-bottom: 1px solid var(--surface-3);
  padding: 0.6rem 1.5rem;
  display: flex;
  gap: 1.5rem;
  align-items: center;
}
.brand {
  font-weight: var(--font-weight-7);
  color: var(--brand);
  margin-right: 1rem;
}
a {
  color: var(--text-2);
  text-decoration: none;
  padding: 0.3rem 0.2rem;
  transition: color var(--ease-3) 0.15s;
}
a:hover {
  color: var(--text-1);
}
a.active {
  color: var(--brand);
  border-bottom: 2px solid var(--brand);
}
</style>
