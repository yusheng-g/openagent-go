import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/sessions':   'http://localhost:8080',  // single-agent API
      '/team/sessions': 'http://localhost:8080',  // team API only
      '/plan/sessions': 'http://localhost:8080',  // plan API only
      '/health':     'http://localhost:8080',
    },
  },
})
