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
/* 内容居中：桌面占 80%（上限 1200px），移动端适配后续在此加媒体查询 */
.content-page {
  width: min(80%, 1200px);
  margin: 0 auto;
  padding: 1.4rem 1.2rem;
}
</style>
