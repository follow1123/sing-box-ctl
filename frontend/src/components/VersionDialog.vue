<script setup lang="ts">
import Modal from './Modal.vue'

export interface VersionItem {
  version: string
  label: string
  isCurrent: boolean
}

defineProps<{ items: VersionItem[]; empty?: string }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'restore', version: string): void }>()
</script>

<template>
  <Modal title="历史版本" @close="emit('close')">
    <p v-if="items.length === 0" class="empty">{{ empty || '暂无版本记录' }}</p>
    <div v-else class="rows">
      <div v-for="it in items" :key="it.version" class="row">
        <span class="name" :class="{ muted: it.isCurrent }">{{ it.label }}</span>
        <div class="ops">
          <span v-if="it.isCurrent" class="cur-tag">当前</span>
          <button v-else class="restore" @click="emit('restore', it.version)">还原</button>
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.empty {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-2);
}
.rows {
  display: flex;
  flex-direction: column;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.2rem;
  border-bottom: 1px solid var(--surface-3);
  font-size: 0.9rem;
}
.row:last-child {
  border-bottom: none;
}
.name {
  color: var(--text-1);
}
.name.muted {
  color: var(--text-2);
}
.ops {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.cur-tag {
  font-size: 0.8rem;
  color: var(--text-2);
}
.restore {
  padding: 0.3rem 0.8rem;
  background: var(--brand);
  color: #fff;
  font-size: 0.85rem;
  border: none;
  border-radius: var(--radius-2);
  cursor: pointer;
}
.restore:hover {
  filter: brightness(0.92);
}
</style>
