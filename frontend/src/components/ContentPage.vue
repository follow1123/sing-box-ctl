<template>
  <div class="content-page">
    <slot />
  </div>
</template>

<style scoped>
/* 内容页统一容器：桌面居中占 80%（上限 1200px），移动端适配后续在此加媒体查询 */
.content-page {
  flex: 1;
  width: min(80%, 1200px);
  margin: 0 auto;
  padding: 1.4rem 1.2rem;
  overflow-y: auto;
}
</style>
