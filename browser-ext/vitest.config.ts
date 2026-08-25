import { defineConfig } from 'vitest/config'

export default defineConfig({
    test: {
        // Vitest owns tests/ only — the Playwright e2e lives in e2e/ and runs
        // through `playwright test`, not vitest.
        include: ['tests/**/*.test.{ts,tsx}'],
        // Pure-logic suites (api, config, background, …) run under node; the
        // popup component tests opt into jsdom per-file with a
        // `// @vitest-environment jsdom` directive. globals lets
        // @testing-library/react auto-cleanup between tests.
        environment: 'node',
        globals: true,
        setupFiles: ['./tests/setup.ts'],
    },
})
