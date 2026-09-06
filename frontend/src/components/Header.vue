<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import ThemeSwitcher from './ThemeSwitcher.vue'
import Btn from './ui/Btn.vue'

const LINKS = [
  { hash: '/', label: 'Provider 管理' },
  { hash: '/templates', label: '模板管理' },
  { hash: '/url', label: '生成 URL' },
]

const active = ref(window.location.hash.slice(1) || '/')
const menuOpen = ref(false)

function onHashChange(): void {
  active.value = window.location.hash.slice(1) || '/'
  menuOpen.value = false
}
onMounted(() => window.addEventListener('hashchange', onHashChange))
onBeforeUnmount(() => window.removeEventListener('hashchange', onHashChange))
</script>

<template>
  <header class="header">
    <div class="row">
      <!-- 手机端：最左汉堡 -->
      <Btn
        variant="ghost"
        size="sm"
        square
        class="burger"
        aria-label="菜单"
        @click="menuOpen = !menuOpen"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M3 6h18v2H3zm0 5h18v2H3zm0 5h18v2H3z" />
        </svg>
      </Btn>

      <!-- 标题 -->
      <span class="brand">SBC</span>

      <!-- 桌面端导航（文本链接，先不做成组件） -->
      <nav class="nav">
        <a
          v-for="l in LINKS"
          :key="l.hash"
          :href="'#' + l.hash"
          :class="{ active: active === l.hash }"
        >
          {{ l.label }}
        </a>
      </nav>

      <!-- 最右：主题切换 -->
      <ThemeSwitcher class="theme" />
    </div>

    <!-- 手机端下拉菜单（导航） -->
    <div v-if="menuOpen" class="menu">
      <a
        v-for="l in LINKS"
        :key="l.hash"
        :href="'#' + l.hash"
        :class="{ active: active === l.hash }"
      >
        {{ l.label }}
      </a>
    </div>
  </header>
</template>

<style scoped>
.header {
  position: relative;
  background: var(--bg);
  border-bottom: 1px solid var(--border);
}
/* 桌面（≥768）：flex 一行：标题 + 导航在左，主题在右 */
.row {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  padding: 0.5rem 1.5rem;
}
.brand {
  font-weight: var(--font-weight-7);
  font-size: 0.95rem;
  letter-spacing: 0.04em;
  color: var(--text);
}
.nav {
  display: flex;
  align-items: center;
  gap: 1.4rem;
}
.nav a {
  text-decoration: none;
  color: var(--text-secondary);
  font-size: 0.875rem;
  padding: 0.3rem 0.1rem;
  border-bottom: 2px solid transparent;
  transition:
    color var(--transition-fast),
    border-color var(--transition-fast);
}
.nav a:hover {
  color: var(--text);
}
.nav a.active {
  color: var(--accent-strong);
  border-bottom-color: var(--accent-strong);
}
.theme {
  margin-left: auto;
}
.burger {
  display: none;
}
.menu {
  display: none;
}

/* 手机（<768）：grid 三区 —— 左汉堡 / 中标题 / 右主题；导航进下拉 */
@media (max-width: 767px) {
  .row {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 0;
    padding: 0.5rem 0.8rem;
  }
  .burger {
    display: inline-flex;
    justify-self: start;
  }
  .brand {
    justify-self: center;
  }
  .nav {
    display: none;
  }
  .theme {
    margin-left: 0;
    justify-self: end;
  }
  .menu {
    position: absolute;
    top: calc(100% + 6px);
    left: 0.8rem;
    z-index: 200;
    min-width: 180px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0.4rem;
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-overlay);
  }
  .menu a {
    display: flex;
    align-items: center;
    padding: 0.55rem 0.8rem;
    border-radius: var(--radius-md);
    font-size: 0.875rem;
    color: var(--text);
    text-decoration: none;
  }
  .menu a:hover {
    background: var(--hover);
  }
  .menu a.active {
    color: var(--accent-strong);
    background: var(--accent-soft);
  }
}
</style>
