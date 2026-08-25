export const DEFAULT_HOST = 'http://127.0.0.1'
// The ports the extension probes in order: thoth serve on 8333, then make dev
// on 8334. The popup's server URL input overrides both.
export const DEFAULT_PORTS = [8333, 8334] as const
export const DEFAULT_BASE_URLS: readonly string[] = DEFAULT_PORTS.map((port) => `${DEFAULT_HOST}:${port}`)

// probeServer reports whether a Thoth server answers /api/v1/health at baseUrl
// within timeoutMs. MV3 host_permissions let the fetch bypass CORS, so no
// server CORS changes are needed for the extension.
export async function probeServer(baseUrl: string, timeoutMs = 1500): Promise<boolean> {
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), timeoutMs)
    try {
        const res = await fetch(`${baseUrl}/api/v1/health`, { signal: controller.signal })
        return res.ok
    } catch {
        return false
    } finally {
        clearTimeout(timer)
    }
}

// discoverBaseUrl returns the first candidate that answers health, or null.
export async function discoverBaseUrl(candidates: readonly string[]): Promise<string | null> {
    for (const base of candidates) {
        if (await probeServer(base)) return base
    }
    return null
}
