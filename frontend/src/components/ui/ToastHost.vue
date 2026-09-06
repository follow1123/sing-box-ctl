<script setup lang="ts">
import { toasts, dismiss } from '../../composables/useToast'
import Btn from './Btn.vue'
</script>

<template>
  <Teleport to="body">
    <div class="toast-host" aria-live="polite">
      <TransitionGroup name="toast">
        <div v-for="t in toasts" :key="t.id" class="toast" :class="t.type" role="alert">
          <!-- 普通：信息圆标 -->
          <svg v-if="t.type === 'info'" class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <circle cx="12" cy="12" r="9" />
            <path d="M12 8h.01" />
            <path d="M11 12h1v4h1" />
          </svg>
          <!-- 警告：三角感叹号 -->
          <svg v-else-if="t.type === 'warn'" class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M12 3l9 16H3z" />
            <path d="M12 9v4" />
            <path d="M12 16.5h.01" />
          </svg>
          <!-- 危险：圆感叹号 -->
          <svg v-else class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <circle cx="12" cy="12" r="9" />
            <path d="M12 7v6" />
            <path d="M12 16.5h.01" />
          </svg>

          <span class="msg">{{ t.message }}</span>
          <Btn
            variant="ghost"
            size="sm"
            square
            class="close"
            aria-label="关闭"
            @click="dismiss(t.id)"
          >
            ✕
          </Btn>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-host {
  position: fixed;
  right: 1rem;
  bottom: 1rem;
  z-index: 500;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  width: min(360px, calc(100vw - 2rem));
  pointer-events: none;
}
.toast {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 0.55rem;
  padding: 0.7rem 2.4rem 0.7rem 0.85rem;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--text);
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-overlay);
  pointer-events: auto;
}
/* 三型差异：不透明卡底 + 语义色描边/图标 */
.toast.info {
  border-color: var(--info);
}
.toast.warn {
  border-color: var(--medium);
}
.toast.error {
  border-color: var(--danger);
}
.icon {
  flex-shrink: 0;
  width: 1rem;
  height: 1rem;
  margin-top: 2px;
}
.toast.info .icon {
  color: var(--info);
}
.toast.warn .icon {
  color: var(--medium);
}
.toast.error .icon {
  color: var(--danger);
}
.msg {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}
.toast .close {
  position: absolute;
  top: 0.4rem;
  right: 0.4rem;
  width: 1.4rem !important;
  height: 1.4rem !important;
  font-size: 0.8rem;
}

/* 动画（open-props keyframes） */
.toast-enter-active {
  animation: slide-in-right 0.25s var(--ease-out-3) both;
}
.toast-leave-active {
  animation: slide-out-right 0.22s var(--ease-out-3) both;
}
</style>
