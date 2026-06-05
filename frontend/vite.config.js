import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Dev server proxies /api to the Go backend so the frontend uses same-origin calls.
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
