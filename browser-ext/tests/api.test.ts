import { afterEach, describe, expect, it, vi } from 'vitest'
import { ThothClient, ThothError } from '../src/core/api'

function stubFetch(handler: (url: string, init?: RequestInit) => Promise<unknown>) {
    const impl = vi.fn(async (url: string, init?: RequestInit) => {
        const body = await handler(url, init)
        return {
            ok: true,
            status: 200,
            json: async () => body,
        }
    })
    vi.stubGlobal('fetch', impl)
    return impl
}

function jsonResponse(body: unknown, status = 200) {
    return {
        ok: status >= 200 && status < 300,
        status,
        json: async () => body,
    }
}

afterEach(() => {
    vi.unstubAllGlobals()
})

const client = () => new ThothClient('http://127.0.0.1:8333')

describe('ThothClient.capture', () => {
    it('POSTs the capture body and returns the parsed response', async () => {
        const fetchMock = stubFetch(async (url, init) => {
            expect(url).toBe('http://127.0.0.1:8333/api/v1/capture')
            expect(init?.method).toBe('POST')
            const body = JSON.parse((init?.body as string | undefined) ?? '{}') as { kind: string }
            expect(body.kind).toBe('bookmark')
            return { path: 'links/bookmarks.md', title: 'A', type: 'bookmark' }
        })
        const res = await client().capture({ kind: 'bookmark', url: 'https://example.com/a', title: 'A' })
        expect(res.path).toBe('links/bookmarks.md')
        expect(fetchMock).toHaveBeenCalledOnce()
    })

    it('throws a ThothError carrying the 409 path for duplicates', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => jsonResponse({ error: 'url already saved', path: 'links/bookmarks.md' }, 409)))
        const err = await client().capture({ kind: 'bookmark', url: 'https://example.com/a', title: 'A' }).catch((e: unknown) => e)
        expect(err).toBeInstanceOf(ThothError)
        if (err instanceof ThothError) {
            expect(err.status).toBe(409)
            expect(err.path).toBe('links/bookmarks.md')
        }
    })

    it('surfaces a server error message', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => jsonResponse({ error: 'text is required for notes' }, 400)))
        const err = await client().capture({ kind: 'note', text: '' }).catch((e: unknown) => e)
        expect(err).toBeInstanceOf(ThothError)
        if (err instanceof ThothError) {
            expect(err.status).toBe(400)
            expect(err.message).toBe('text is required for notes')
        }
    })

    it('maps a network failure to a reachability error', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => {
            throw new Error('failed to fetch')
        }))
        const err = await client().capture({ kind: 'note', text: 'x' }).catch((e: unknown) => e)
        expect(err).toBeInstanceOf(ThothError)
        if (err instanceof ThothError) expect(err.message).toBe('server not reachable')
    })
})

describe('ThothClient.checkDuplicate', () => {
    it('reports exists=true with the path from the check endpoint', async () => {
        stubFetch(async (url) => {
            expect(url).toContain('/api/v1/capture/check?url=')
            return { exists: true, path: 'links/bookmarks.md' }
        })
        const dup = await client().checkDuplicate('https://example.com/a')
        expect(dup).toEqual({ exists: true, path: 'links/bookmarks.md' })
    })
})

describe('ThothClient.inboxCount', () => {
    it('returns the count from the inbox-count endpoint', async () => {
        stubFetch(async () => ({ count: 3 }))
        expect(await client().inboxCount()).toBe(3)
    })
})

describe('ThothClient.summarize', () => {
    it('POSTs the page to the summarize endpoint and returns the note', async () => {
        stubFetch(async (url, init) => {
            expect(url).toBe('http://127.0.0.1:8333/api/v1/capture/summarize')
            expect(init?.method).toBe('POST')
            return { path: 'knowledge/summary-a.md', title: 'Summary: A', type: 'knowledge' }
        })
        const res = await client().summarize({ url: 'https://example.com/a', title: 'A', text: 'page' })
        expect(res.path).toBe('knowledge/summary-a.md')
    })
})

describe('ThothClient.folders', () => {
    it('returns the configured folders from settings', async () => {
        stubFetch(async () => ({ wiki_folders: ['inbox', 'knowledge'] }))
        expect(await client().folders()).toEqual(['inbox', 'knowledge'])
    })
})
