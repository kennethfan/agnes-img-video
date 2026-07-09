import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        configure(proxy) {
          proxy.on('proxyReq', (proxyReq, req) => {
            const ip = req.headers['x-forwarded-for'] ?? req.socket?.remoteAddress ?? ''
            if (ip) {
              proxyReq.setHeader('X-Forwarded-For', Array.isArray(ip) ? ip[0] : ip)
            }
          })
        },
      },
      '/outputs': 'http://localhost:8080',
    },
  },
})
