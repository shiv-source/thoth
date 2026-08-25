import { afterEach, describe, expect, it, vi } from 'vitest'
import { ThothClient } from '../src/core/api'
import { refreshBadge } from '../src/core/badge'
import { fakeBrowserAPI } from './fakes'

function stubFetch(body: unknown, ok = true) {
    vi.stubGlobal(
        'fetch',
        vi.fn(async () => ({
            ok,
            status: ok ? 200 : 500,
            json: async () => body,
        }))
    )
}

afterEach(() => {
    vi.unstubAllGlobals()
})

describe('refreshBadge', () => {
    it('sets the badge to the inbox count', async () => {
        stubFetch({ count: 3 })
        const fake = fakeBrowserAPI()
        await refreshBadge(fake.api, new ThothClient('http://127.0.0.1:8333'))
        expect(fake.badge).toContainEqual({ type: 'text', value: '3' })
        expect(fake.badge).toContainEqual({ type: 'color', value: '#3b82f6' })
    })

    it('clears the badge when there are no unfiled captures', async () => {
        stubFetch({ count: 0 })
        const fake = fakeBrowserAPI()
        await refreshBadge(fake.api, new ThothClient('http://127.0.0.1:8333'))
        expect(fake.badge).toContainEqual({ type: 'text', value: '' })
    })

    it('clears the badge silently when the server is unreachable', async () => {
        stubFetch({}, false)
        const fake = fakeBrowserAPI()
        await refreshBadge(fake.api, new ThothClient('http://127.0.0.1:8333'))
        expect(fake.badge).toContainEqual({ type: 'text', value: '' })
    })
})
