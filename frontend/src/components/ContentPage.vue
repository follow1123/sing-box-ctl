<template>
  <!-- 页面内容宽度容器：滚动由 App 的 main 统一负责，这里只做居中与断点宽度 -->
  <div class="content-page">
    <slot />
  </div>
</template>

<style scoped>
/* 手机（<768px）：约 92%；平板（768-1024px）：占满；桌面（>1024px）：80% 居中 */
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
</style>
