import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// 手机摄像头（扫码核销）要求安全上下文：HTTPS 或 localhost。
// 存在 scripts/certs/dev-cert.pem 时 dev server 以 HTTPS 启动（自签名，首次访问需信任）。
const here = path.dirname(fileURLToPath(import.meta.url))
const certDir = path.resolve(here, '../../scripts/certs')
const keyFile = path.join(certDir, 'dev-key.pem')
const certFile = path.join(certDir, 'dev-cert.pem')
const https =
  fs.existsSync(keyFile) && fs.existsSync(certFile)
    ? { key: fs.readFileSync(keyFile), cert: fs.readFileSync(certFile) }
    : undefined

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    // 商家端 dev server：监听所有网卡，/api 与 /admin 代理到 Go 后端
    host: true,
    port: 5174,
    allowedHosts: true,
    https,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/admin': 'http://127.0.0.1:8080',
    },
  },
})
