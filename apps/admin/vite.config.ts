import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// 手机摄像头（扫码核销）要求安全上下文：HTTPS 或 localhost。
// `vite --mode https` 时以 HTTPS 启动在 5175（手机扫码专用，自签名证书，首次需信任）；
// 默认 HTTP 启动在 5174（电脑/内置浏览器日常使用）。
const here = path.dirname(fileURLToPath(import.meta.url))
const certDir = path.resolve(here, '../../scripts/certs')
const keyFile = path.join(certDir, 'dev-key.pem')
const certFile = path.join(certDir, 'dev-cert.pem')

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const httpsMode = mode === 'https'
  return {
    plugins: [vue()],
    server: {
      // 商家端 dev server：监听所有网卡，/api 与 /admin 代理到 Go 后端
      host: true,
      port: httpsMode ? 5175 : 5174,
      allowedHosts: true,
      https:
        httpsMode && fs.existsSync(keyFile) && fs.existsSync(certFile)
          ? { key: fs.readFileSync(keyFile), cert: fs.readFileSync(certFile) }
          : undefined,
      proxy: {
        '/api': 'http://127.0.0.1:8080',
        '/admin': 'http://127.0.0.1:8080',
      },
    },
  }
})
