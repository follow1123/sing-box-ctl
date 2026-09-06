<script setup lang="ts">
import { ref, computed } from 'vue'
import Header from './components/Header.vue'
import ToastHost from './components/ui/ToastHost.vue'
import ProviderView from './views/ProviderView.vue'
import TemplateView from './views/TemplateView.vue'
import UrlView from './views/UrlView.vue'

// 简单 hash 路由（三个页面）
const route = ref(window.location.hash.slice(1) || '/')
window.addEventListener('hashchange', () => {
  route.value = window.location.hash.slice(1) || '/'
})
const page = computed(() => route.value)
</script>

<template>
  <div class="app">
    <header class="app-header">
      <Header :active="page" />
    </header>

    <!-- 内容滚动区：所有页面共用，内容过长时滚动条贴窗口最右 -->
    <main class="app-main">
      <ProviderView v-if="page === '/'" />
      <TemplateView v-else-if="page === '/templates'" />
      <UrlView v-else />
    </main>

    <!-- 全局通知：右下角最多 3 条 -->
    <ToastHost />
  </div>
</template>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.app-header {
  flex-shrink: 0;
}
.app-main {
  flex: 1;
  min-height: 0;
  width: 100%;
  overflow-y: auto;
}
</style>
