<script setup lang="ts">
/** 页面两栏布局：左侧栏 + 右侧内容区 */
defineProps<{ asideTitle?: string }>()
</script>

<template>
  <main>
    <aside>
      <h2 v-if="asideTitle">{{ asideTitle }}</h2>
      <slot name="aside" />
    </aside>
    <div class="content">
      <slot />
    </div>
  </main>
</template>

<style scoped>
main {
  flex: 1;
  display: flex;
  overflow: hidden;
}
aside {
  width: 260px;
  border-right: 1px solid var(--surface-3);
  background: var(--surface-2);
  padding: 1rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
aside :slotted(h2) {
  font-size: 0.9rem;
  margin: 0 0 0.5rem;
  color: var(--text-2);
}
.content {
  flex: 1;
  padding: 1.2rem;
  overflow-y: auto;
}
</style>
