<script setup lang="ts">
defineProps<{ title?: string }>()
</script>

<template>
  <section class="card">
    <header v-if="title || $slots.actions" class="card-header">
      <h3 v-if="title" class="card-title">{{ title }}</h3>
      <div v-if="$slots.actions" class="card-actions">
        <slot name="actions" />
      </div>
    </header>
    <slot />
  </section>
</template>

<style scoped>
.card {
  padding: 1rem;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-card);
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.6rem;
}
.card-title {
  margin: 0;
  font-size: 0.95rem;
  font-weight: var(--font-weight-6);
  color: var(--text);
}
.card-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
</style>
