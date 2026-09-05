<template>
  <!-- 滚动区占满视口宽度，滚动条贴在窗口最右；内容在内部居中 -->
  <div class="scroll-area">
    <div class="content-page">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.scroll-area {
  flex: 1;
  width: 100%;
  overflow-y: auto;
}
/* 手机（<768px）：主内容 ~92%；平板（768-1024px）：占满；桌面（>1024px）：80% 居中 */
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
