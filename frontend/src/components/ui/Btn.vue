<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 视觉变体 */
    variant?: 'default' | 'primary' | 'danger' | 'soft' | 'ghost'
    /** 尺寸：md=常规控件高，sm=紧凑（行内小按钮） */
    size?: 'md' | 'sm'
    type?: 'button' | 'submit'
    disabled?: boolean
    title?: string
    /** 方形图标按钮（宽=高，常用于 ghost） */
    square?: boolean
  }>(),
  { variant: 'default', size: 'md', type: 'button', disabled: false, square: false },
)
const emit = defineEmits<{ (e: 'click', ev: MouseEvent): void }>()

const cls = computed(() => [`btn-${props.variant}`, `btn-${props.size}`, { 'btn-square': props.square }])
</script>

<template>
  <button
    class="btn"
    :class="cls"
    :type="type"
    :disabled="disabled"
    :title="title"
    @click="emit('click', $event)"
  >
    <slot />
  </button>
</template>

<style scoped>
/* 变体样式随 size 走（不在此限定高/内边距） */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  padding: 0 0.9rem;
  font-size: 0.875rem;
  font-weight: var(--font-weight-5);
  line-height: 1;
  white-space: nowrap;
  color: var(--text);
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  box-shadow: var(--shadow-control);
  cursor: pointer;
  transition:
    background var(--transition-fast),
    border-color var(--transition-fast),
    color var(--transition-fast),
    box-shadow var(--transition-fast);
}
.btn:hover {
  background: var(--inset);
}
.btn:disabled {
  opacity: 0.5;
  cursor: default;
  pointer-events: none;
}

/* md：标准控件高；sm：26px 紧凑 */
.btn-md {
  height: var(--control-height);
}
.btn-sm {
  height: 26px;
  padding: 0 0.7rem;
  font-size: 0.8rem;
}
/* 方形（图标按钮） */
.btn-square {
  padding: 0;
}
.btn-md.btn-square {
  width: var(--control-height);
}
.btn-sm.btn-square {
  width: 26px;
}

/* primary：accent 实心 */
.btn-primary {
  background: var(--accent);
  border-color: transparent;
  color: var(--on-accent);
}
.btn-primary:hover {
  background: var(--accent-strong);
}

/* danger：与普通按钮同构（白底细边阴影），文字红色；hover 淡红底 */
.btn-danger {
  color: var(--danger);
}
.btn-danger:hover {
  background: var(--danger-soft);
  color: var(--danger);
}

/* soft：弱强调（默认/选中项等） */
.btn-soft {
  color: var(--accent-strong);
  background: var(--accent-soft);
  border-color: transparent;
}

/* ghost：透明图标按钮（关闭/菜单/工具图标） */
.btn-ghost {
  background: transparent;
  border-color: transparent;
  box-shadow: none;
  color: var(--text-secondary);
}
.btn-ghost:hover {
  background: var(--hover);
  color: var(--text);
}

/* 移动端：加大可点区域（dashboard 做法：control-height 36px） */
@media (max-width: 720px) {
  .btn-md {
    height: 36px;
    padding: 0 1.1rem;
    font-size: 0.95rem;
  }
  .btn-sm {
    height: 30px;
  }
}
</style>
