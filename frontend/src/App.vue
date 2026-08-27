<script setup lang="ts">
import { ref, computed } from 'vue'
import NavBar from './components/NavBar.vue'
import ProviderView from './views/ProviderView.vue'
import TemplateView from './views/TemplateView.vue'
import UrlView from './views/UrlView.vue'

// 简单 hash 路由（三个静态页面）
const route = ref(window.location.hash.slice(1) || '/')
window.addEventListener('hashchange', () => {
  route.value = window.location.hash.slice(1) || '/'
})
const page = computed(() => route.value)
</script>

<template>
  <NavBar :active="page" />
  <ProviderView v-if="page === '/'" />
  <TemplateView v-else-if="page === '/templates'" />
  <UrlView v-else />
</template>
