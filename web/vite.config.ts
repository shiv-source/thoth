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
    // Async UI tests (fakeWS-backed settings/chat flows) run slower under the
    // v8 coverage instrumentation on shared CI runners; give them headroom
    // past vitest's 5s default so the cap doesn't flake.
    testTimeout: 15000,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary'],
      // Keep reportOnFailure false: the frontend-test coverage gate in
      // quality.yml depends on a missing coverage-summary.json meaning a
      // failed run (vitest writes no coverage files when tests fail).
      reportOnFailure: false,
      // Coverage floors — single source of truth for the frontend gate (the
      // vitest run fails below them) and for the final-gate report's floor
      // line (the coverage gate reads all thresholds here). Keep in sync
      // with the Makefile web-cover target and COVERAGE_FLOOR.
      thresholds: {
        lines: 90,
        functions: 90,
        branches: 80,
        statements: 90,
      },
    },
  },
})
