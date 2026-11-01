import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const nginx = loadEnv(mode, '.', 'MIRROROUC_').MIRROROUC_NGINX_ORIGIN || 'http://127.0.0.1:8080'
  return {
    plugins: [vue()],
    build: { rollupOptions: { input: { main: 'index.html', directory: 'directory.html' } } },
    server: {
      port: 5173, strictPort: true,
      proxy: {
        '/api': 'http://127.0.0.1:8081', '/uploads': 'http://127.0.0.1:8081',
        '/mirrors': nginx,
        // The server-rendered directory references its own built asset hashes.
        '/assets/': nginx,
      },
    },
  }
})
