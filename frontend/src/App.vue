<script setup lang="ts">
import Header from './components/Header.vue'
import ProviderView from './views/ProviderView.vue'
import ToastHost from './components/ui/ToastHost.vue'

// 应用骨架：header 固定，main 为唯一滚动区。当前先展示 Provider 列表页，
// 其余页面后续逐个接入。
</script>

<template>
  <div class="app">
    <header class="app-header">
      <Header />
    </header>

    <!-- 内容滚动区：所有页面共用，内容过长时滚动条贴窗口最右 -->
    <main class="app-main">
      <ProviderView />
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
