import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiProxyUrl = env.VITE_API_PROXY_URL || 'http://localhost:8080'
  const port = Number(env.VITE_PORT || 5173)

  return {
    plugins: [react(), tailwindcss()],
    server: {
      port,
      host: true,
      proxy: {
        '/api': {
          target: apiProxyUrl,
          changeOrigin: true,
        },
      },
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
  }
})
