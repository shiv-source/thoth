import { afterEach, describe, expect, it, vi } from 'vitest'
import { connectBaseUrl, ensureHostPermission, resolveBaseUrl, saveBaseUrl, BASE_URL_KEY } from '../src/core/config'
import { DEFAULT_BASE_URLS, discoverBaseUrl, probeServer } from '../src/core/server'
import type { BrowserAPI } from '../src/core/webext'
import { memoryStorage } from './fakes'

function stubFetch(handler: (url: string, init?: RequestInit) => Promise<{ ok: boolean }>) {
    const impl = vi.fn(async (url: string, init?: RequestInit) => handler(url, init))
    vi.stubGlobal('fetch', impl)
    return impl
}

afterEach(() => {
    vi.unstubAllGlobals()
})

describe('server discovery', () => {
    it('probes the default ports in order and returns the first live one', async () => {
        const fetchMock = stubFetch(async (url) => ({ ok: url.includes(':8334') }))
        const base = await discoverBaseUrl(DEFAULT_BASE_URLS)
        expect(base).toBe('http://127.0.0.1:8334')
        // 8333 was probed first (returned ok:false), then 8334.
        expect(fetchMock.mock.calls[0]?.[0]).toContain(':8333')
        expect(fetchMock.mock.calls[1]?.[0]).toContain(':8334')
    })

    it('returns null when no server answers', async () => {
        stubFetch(async () => ({ ok: false }))
        expect(await discoverBaseUrl(DEFAULT_BASE_URLS)).toBeNull()
    })

    it('treats a fetch failure as down', async () => {
        stubFetch(async () => {
            throw new Error('refused')
        })
        expect(await probeServer('http://127.0.0.1:9999')).toBe(false)
    })
})

describe('resolveBaseUrl', () => {
    it('prefers a reachable custom base URL', async () => {
        const store = memoryStorage({ [BASE_URL_KEY]: 'http://127.0.0.1:9000/' })
        stubFetch(async (url) => ({ ok: url.includes(':9000') }))
        expect(await resolveBaseUrl(store)).toBe('http://127.0.0.1:9000')
    })

    it('falls back to the default ports when the custom URL is stale', async () => {
        const store = memoryStorage({ [BASE_URL_KEY]: 'http://127.0.0.1:9000' })
        stubFetch(async (url) => ({ ok: url.includes(':8333') }))
        expect(await resolveBaseUrl(store)).toBe('http://127.0.0.1:8333')
    })

    it('discovers the defaults when no custom URL is stored', async () => {
        const store = memoryStorage()
        stubFetch(async (url) => ({ ok: url.includes(':8334') }))
        expect(await resolveBaseUrl(store)).toBe('http://127.0.0.1:8334')
    })
})

describe('saveBaseUrl', () => {
    it('stores a normalized custom URL', async () => {
        const store = memoryStorage()
        await saveBaseUrl(store, ' http://127.0.0.1:9000/ ')
        expect(store.data[BASE_URL_KEY]).toBe('http://127.0.0.1:9000')
    })

    it('clears the custom URL on empty input', async () => {
        const store = memoryStorage({ [BASE_URL_KEY]: 'http://127.0.0.1:9000' })
        await saveBaseUrl(store, '   ')
        expect(store.data[BASE_URL_KEY]).toBeUndefined()
    })
})

describe('connectBaseUrl', () => {
    it('returns a reachable custom URL', async () => {
        stubFetch(async (url) => ({ ok: url.includes(':9000') }))
        expect(await connectBaseUrl(' http://127.0.0.1:9000/ ')).toBe('http://127.0.0.1:9000')
    })

    it('does not fall back to the default ports when a custom URL is down', async () => {
        const fetchMock = stubFetch(async () => ({ ok: false }))
        expect(await connectBaseUrl('http://127.0.0.1:9000')).toBeNull()
        // The custom URL was the only thing probed — 8333/8334 were never tried.
        expect(fetchMock.mock.calls.map((c) => c[0])).toEqual(['http://127.0.0.1:9000/api/v1/health'])
    })

    it('discovers the defaults when nothing is entered', async () => {
        stubFetch(async (url) => ({ ok: url.includes(':8334') }))
        expect(await connectBaseUrl('')).toBe('http://127.0.0.1:8334')
    })
})

describe('ensureHostPermission', () => {
    it('is a no-op for the statically permitted localhost hosts', async () => {
        const request = vi.fn(async () => false) // a denial would be ignored — never called
        const api = { permissions: { request } } as unknown as BrowserAPI
        await expect(ensureHostPermission(api, 'http://127.0.0.1:9000')).resolves.toBe(true)
        await expect(ensureHostPermission(api, 'http://localhost:8333')).resolves.toBe(true)
        expect(request).not.toHaveBeenCalled()
    })

    it('requests a permission grant for a non-localhost origin', async () => {
        const request = vi.fn(async () => true)
        const api = { permissions: { request } } as unknown as BrowserAPI
        await expect(ensureHostPermission(api, 'http://192.168.1.5:8333')).resolves.toBe(true)
        expect(request).toHaveBeenCalledWith({ origins: ['http://192.168.1.5:8333/*'] })
    })

    it('returns false when the user denies the grant', async () => {
        const api = { permissions: { request: vi.fn(async () => false) } } as unknown as BrowserAPI
        await expect(ensureHostPermission(api, 'http://host:8333')).resolves.toBe(false)
    })

    it('returns false when the browser has no permissions API', async () => {
        await expect(ensureHostPermission({} as BrowserAPI, 'http://host:8333')).resolves.toBe(false)
    })

    it('returns true for empty input (auto-discovery on localhost)', async () => {
        await expect(ensureHostPermission({} as BrowserAPI, '')).resolves.toBe(true)
    })
})
