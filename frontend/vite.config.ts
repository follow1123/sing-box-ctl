import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    // 构建产物输出到 Go 包目录，供 go:embed
    outDir: '../webui/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    // 开发模式代理到 Go 服务（本地起 sbctl serve -p 9112）
    proxy: {
      '/api': 'http://127.0.0.1:9112',
      '/config': 'http://127.0.0.1:9112',
    },
  },
})
