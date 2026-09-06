import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    // 产物输出到仓库根 webui/dist，供 Go 侧 //go:embed dist 编译进二进制
    outDir: '../webui/dist',
    emptyOutDir: true,
  },
})
