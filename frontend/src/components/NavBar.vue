<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import ThemeSwitcher from './ThemeSwitcher.vue'

defineProps<{ active: string }>()

const menuOpen = ref(false)

function onHashChange(): void {
  menuOpen.value = false
}

onMounted(() => window.addEventListener('hashchange', onHashChange))
onBeforeUnmount(() => window.removeEventListener('hashchange', onHashChange))
</script>

<template>
  <nav>
    <!-- 手机端：汉堡菜单 -->
    <button class="burger" aria-label="菜单" @click="menuOpen = !menuOpen">
      <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path d="M3 6h18v2H3zm0 5h18v2H3zm0 5h18v2H3z" />
      </svg>
    </button>
    <div v-if="menuOpen" class="menu">
      <a href="#/" :class="{ active: active === '/' }">Provider 管理</a>
      <a href="#/templates" :class="{ active: active === '/templates' }">模板管理</a>
      <a href="#/url" :class="{ active: active === '/url' }">生成 URL</a>
    </div>

    <!-- 桌面端：标题 + 文字导航 -->
    <span class="brand">sbctl</span>
    <a class="nav-link" href="#/" :class="{ active: active === '/' }">Provider 管理</a>
    <a class="nav-link" href="#/templates" :class="{ active: active === '/templates' }">模板管理</a>
    <a class="nav-link" href="#/url" :class="{ active: active === '/url' }">生成 URL</a>

    <ThemeSwitcher />
  </nav>
</template>

<style scoped>
nav {
  position: relative;
  background: var(--surface-2);
  border-bottom: 1px solid var(--surface-3);
  padding: 0.55rem 1.5rem;
  display: flex;
  gap: 1.4rem;
  align-items: center;
}
.brand {
  font-weight: var(--font-weight-7);
  color: var(--brand);
  margin-right: 0.4rem;
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
.burger {
  display: none;
}

/* 手机端：汉堡 + 下拉菜单，隐藏标题与文字导航 */
@media (max-width: 767px) {
  nav {
    gap: 0.4rem;
    padding: 0.5rem 0.8rem;
  }
  .brand,
  .nav-link {
    display: none;
  }
  .burger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.9rem;
    height: 1.9rem;
    padding: 0;
    background: none;
    border: none;
    border-radius: var(--radius-1);
    color: var(--text-2);
    cursor: pointer;
  }
  .burger:hover {
    background: var(--surface-3);
    color: var(--text-1);
  }
  .burger svg {
    width: 1.2rem;
    height: 1.2rem;
  }
  .menu {
    position: absolute;
    top: calc(100% + 2px);
    left: 0.8rem;
    z-index: 200;
    min-width: 170px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0.4rem;
    background: var(--surface-2);
    border: 1px solid var(--surface-3);
    border-radius: var(--radius-2);
    box-shadow: var(--shadow-3);
  }
  .menu a {
    padding: 0.55rem 0.8rem;
    border-radius: var(--radius-1);
  }
  .menu a:hover {
    background: var(--surface-3);
  }
  .menu a.active {
    color: var(--brand);
    border-bottom: none;
    background: var(--surface-3);
  }
}
</style>
