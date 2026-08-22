import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// make dev serves on the dev port (serve --dev) and sets THOTH_PORT so the
// proxy follows; standalone pnpm dev proxies to a running instance on 8333.
const apiPort = process.env.THOTH_PORT ?? '8333'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': `http://127.0.0.1:${apiPort}`,
      '/ws': { target: `ws://127.0.0.1:${apiPort}`, ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.tsx',
  },
})
