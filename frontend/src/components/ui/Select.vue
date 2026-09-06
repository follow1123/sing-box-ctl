<script setup lang="ts">
defineProps<{ modelValue?: string; placeholder?: string; disabled?: boolean }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()
</script>

<template>
  <!-- 外观由 tokens.css 元素级 select 基础统一，此处只做双向绑定与占位 -->
  <select
    class="select"
    :value="modelValue"
    :disabled="disabled"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <option v-if="placeholder && !modelValue" value="" disabled>{{ placeholder }}</option>
    <slot />
  </select>
</template>
