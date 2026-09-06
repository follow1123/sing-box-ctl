<script setup lang="ts">
import Modal from './Modal.vue'
import Btn from './ui/Btn.vue'

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
          <Btn v-else size="sm" @click="emit('restore', it.version)">还原</Btn>
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.empty {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-faint);
}
.rows {
  display: flex;
  flex-direction: column;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.55rem 0;
  border-bottom: 1px solid var(--border);
  font-size: 0.9rem;
}
.row:last-child {
  border-bottom: none;
}
.name {
  color: var(--text);
}
.name.muted {
  color: var(--text-faint);
}
.ops {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.cur-tag {
  padding: 2px 8px;
  font-size: 0.75rem;
  color: var(--accent-strong);
  background: var(--accent-soft);
  border-radius: var(--radius-pill);
}
</style>
