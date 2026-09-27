import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    // 商家端 dev server：监听所有网卡，/api 与 /admin 代理到 Go 后端
    host: true,
    port: 5174,
    allowedHosts: true,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/admin': 'http://127.0.0.1:8080',
    },
  },
})
