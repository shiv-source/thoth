import { afterEach, describe, expect, it, vi } from 'vitest'
import { initBackground } from '../src/core/background'
import { DRAFT_KEY } from '../src/core/storage'
import { MENU_BOOKMARK, MENU_SUMMARIZE } from '../src/core/menus'
import { fakeBrowserAPI, memoryStorage } from './fakes'

function stubServer(count = 2) {
    vi.stubGlobal(
        'fetch',
        vi.fn(async (url: string) => {
            if (url.includes('/api/v1/capture/inbox-count')) {
                return { ok: true, status: 200, json: async () => ({ count }) }
            }
            return { ok: true, status: 200, json: async () => ({ status: 'ok' }) }
        })
    )
}

afterEach(() => {
    vi.unstubAllGlobals()
})

function setup() {
    const fake = fakeBrowserAPI()
    const storage = memoryStorage()
    initBackground({ api: fake.api, storage })
    return { fake, storage }
}

describe('initBackground', () => {
    it('registers menus, schedules the badge alarm, and paints the badge on install', async () => {
        stubServer()
        const { fake } = setup()
        await fake.fireInstalled()
        await vi.waitFor(() => expect(fake.menuCreated.length).toBe(4))
        expect(fake.alarmNames).toContain('thoth-badge')
        await vi.waitFor(() => expect(fake.badge.some((b) => b.type === 'text' && b.value === '2')).toBe(true))
    })

    it('stores a bookmark draft and opens the popup on a menu click', async () => {
        stubServer()
        const { fake, storage } = setup()
        await fake.fireMenuClick({ menuItemId: MENU_BOOKMARK, pageUrl: 'https://example.com/a' }, { title: 'A page' })
        await vi.waitFor(() => expect(fake.openedPopup).toBe(true))
        const draft = JSON.parse(storage.data[DRAFT_KEY] ?? '') as { kind: string; url: string; title: string }
        expect(draft.kind).toBe('bookmark')
        expect(draft.url).toBe('https://example.com/a')
        expect(draft.title).toBe('A page')
    })

    it('grabs page text for the summarize menu so the popup can send it', async () => {
        stubServer()
        const { fake, storage } = setup()
        await fake.fireMenuClick({ menuItemId: MENU_SUMMARIZE, pageUrl: 'https://example.com/a' }, { id: 1, title: 'A' })
        await vi.waitFor(() => expect(fake.scripted).toEqual([1]))
        const draft = JSON.parse(storage.data[DRAFT_KEY] ?? '') as { kind: string; text: string }
        expect(draft.kind).toBe('summarize')
        expect(draft.text).toBe('page inner text')
    })

    it('captures the active tab into a bookmark draft via the command', async () => {
        stubServer()
        const { fake, storage } = setup()
        await fake.fireCommand('capture-page')
        await vi.waitFor(() => expect(storage.data[DRAFT_KEY]).toBeDefined())
        expect(fake.queried).toBe(true)
        const draft = JSON.parse(storage.data[DRAFT_KEY] ?? '') as { kind: string; url: string }
        expect(draft.kind).toBe('bookmark')
        expect(draft.url).toBe('https://example.com')
        expect(fake.openedPopup).toBe(true)
    })

    it('hints via the badge when the browser cannot open the popup programmatically', async () => {
        stubServer()
        const { fake, storage } = setup()
        fake.api.action.openPopup = undefined
        await fake.fireMenuClick({ menuItemId: MENU_BOOKMARK, pageUrl: 'https://example.com/a' })
        await vi.waitFor(() =>
            expect(fake.badge.some((b) => b.type === 'text' && b.value === '★')).toBe(true)
        )
        expect(storage.data[DRAFT_KEY]).toBeDefined()
        expect(fake.openedPopup).toBe(false)
    })

    it('refreshes the badge when the scheduled alarm fires', async () => {
        stubServer(5)
        const { fake } = setup()
        await fake.fireInstalled()
        await fake.fireAlarm('thoth-badge')
        await vi.waitFor(() => expect(fake.badge.some((b) => b.type === 'text' && b.value === '5')).toBe(true))
    })
})
