<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'

defineProps<{ title: string }>()
const emit = defineEmits<{ (e: 'close'): void }>()

function onKey(e: KeyboardEvent): void {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="mask" @click.self="emit('close')">
    <div class="dialog" role="dialog" aria-modal="true">
      <div class="header">
        <span class="title">{{ title }}</span>
        <button class="close" type="button" aria-label="关闭" @click="emit('close')">✕</button>
      </div>
      <div class="body">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  background: rgb(0 0 0 / 45%);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.dialog {
  width: min(480px, 92vw);
  max-height: 88vh;
  overflow-y: auto;
  background: var(--surface-2);
  border: 1px solid var(--surface-4);
  border-radius: var(--radius-3);
  box-shadow: var(--shadow-4);
  padding: 1rem 1.2rem;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.8rem;
}
.title {
  font-weight: var(--font-weight-6);
  color: var(--text-1);
}
.close {
  background: none;
  border: none;
  font-size: 1rem;
  cursor: pointer;
  color: var(--text-2);
  padding: 2px 6px;
  border-radius: var(--radius-2);
}
.close:hover {
  background: var(--surface-3);
}
</style>
