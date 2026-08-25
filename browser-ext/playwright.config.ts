import { defineConfig } from '@playwright/test'

// The extension e2e: loads the built MV3 extension into a persistent Chromium
// context and drives the React popup against the mock capture server (see
// e2e/mock-server.mjs) on the fixed port below. The mock implements the same
// capture API contract the Go server exposes, so these tests exercise the
// popup's real browser behavior without needing a Go server in CI.
const MOCK_PORT = 8337

export default defineConfig({
    testDir: './e2e',
    // Extension contexts are heavy and share one profile — run serially.
    workers: 1,
    fullyParallel: false,
    timeout: 60000,
    expect: { timeout: 10000 },
    webServer: {
        command: 'node e2e/mock-server.mjs',
        url: `http://127.0.0.1:${MOCK_PORT}/api/v1/health`,
        reuseExistingServer: false,
        stdout: 'pipe',
        stderr: 'pipe',
    },
    use: {
        trace: 'retain-on-failure',
    },
})
