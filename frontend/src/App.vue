<script setup lang="ts">
// 应用骨架（开发中）：上方 header 区与下方内容区目前均为占位。
</script>

<template>
  <div class="app">
    <!-- 上方区域：后续接入 header 组件（当前为占位） -->
    <header class="app-header">
      <p class="placeholder">Header 区域 — 后续接入 header 组件</p>
    </header>

    <!-- 下方内容区：唯一滚动容器，内容过长时滚动条贴窗口最右 -->
    <main class="app-main">
      <div class="content-page">
        <p class="placeholder">
          内容区域 — 后续渲染各页面（Provider / 模板 / 生成 URL）
        </p>
        <p class="placeholder small">
          宽度断点：手机 &lt;768px 约 92% · 平板 768–1024px 占满 · 桌面 &gt;1024px 80% 居中。
          内容超出视口高度时，滚动发生在本区域（最外层），而非页面内部。
        </p>

        <!-- 占位正文：撑高以验证“滚动条在最外层” -->
        <div class="lorem">
          <p v-for="i in 30" :key="i">
            占位段落 {{ i }} — 用于验证内容区域超长时的外层滚动表现。
          </p>
        </div>
      </div>
    </main>
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
  border-bottom: 1px solid var(--border);
}
.app-main {
  flex: 1;
  min-height: 0;
  width: 100%;
  overflow-y: auto;
}
/* 内容宽度容器占位（后续由页面内容组件承载同一断点逻辑） */
.content-page {
  width: min(92%, 1200px);
  margin: 0 auto;
  padding: 1.1rem 1rem;
}
@media (min-width: 768px) and (max-width: 1024px) {
  .content-page {
    width: 100%;
  }
}
@media (min-width: 1025px) {
  .content-page {
    width: min(80%, 1200px);
  }
}
.placeholder {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
}
.app-header .placeholder {
  padding: 0.6rem 1.5rem;
}
.placeholder.small {
  margin-top: 0.4rem;
  font-size: 0.8rem;
  color: var(--text-faint);
}
.lorem {
  margin-top: 1rem;
  color: var(--text-faint);
  font-size: 0.85rem;
}
.lorem p {
  margin: 0.3rem 0;
}
</style>
