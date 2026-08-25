import { afterEach, describe, expect, it, vi } from 'vitest'
import { resolveBaseUrl, saveBaseUrl, BASE_URL_KEY } from '../src/core/config'
import { DEFAULT_BASE_URLS, discoverBaseUrl, probeServer } from '../src/core/server'
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
