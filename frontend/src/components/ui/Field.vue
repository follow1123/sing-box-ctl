<script setup lang="ts">
defineProps<{ label?: string }>()
</script>

<template>
  <div class="field">
    <label v-if="label" class="field-label">{{ label }}</label>
    <slot />
  </div>
</template>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  margin-bottom: 0.7rem;
}
.field:last-child {
  margin-bottom: 0;
}
.field-label {
  margin-bottom: 4px;
  font-size: 0.8125rem;
  color: var(--text-secondary);
}
</style>
