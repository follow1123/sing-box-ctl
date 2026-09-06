<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'
import Btn from './ui/Btn.vue'

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
        <Btn variant="ghost" size="sm" square aria-label="关闭" @click="emit('close')">✕</Btn>
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
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--scrim);
  z-index: 100;
}
.dialog {
  width: min(520px, 92vw);
  max-height: 85vh;
  overflow-y: auto;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-overlay);
  padding: var(--size-5);
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.6rem;
}
.title {
  font-weight: var(--font-weight-6);
  font-size: 1rem;
  color: var(--text);
}
</style>
