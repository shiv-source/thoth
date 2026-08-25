import { describe, expect, it } from 'vitest'
import { MENU_BOOKMARK, MENU_READLATER, MENU_SELECTION, MENU_SUMMARIZE, draftForMenu, registerMenus } from '../src/core/menus'
import { clearDraft, loadDraft, saveDraft } from '../src/core/storage'
import type { Draft } from '../src/core/types'
import { fakeBrowserAPI, memoryStorage } from './fakes'

describe('registerMenus', () => {
    it('installs the four capture menus', async () => {
        const fake = fakeBrowserAPI()
        await registerMenus(fake.api)
        expect(fake.menuCreated.map((m) => m.id)).toEqual([
            MENU_BOOKMARK,
            MENU_READLATER,
            MENU_SELECTION,
            MENU_SUMMARIZE,
        ])
    })
})

describe('draftForMenu', () => {
    it('bookmarks a page with metadata only', () => {
        const draft = draftForMenu(MENU_BOOKMARK, { url: 'https://example.com/a', title: 'A page' })
        expect(draft).toEqual({ kind: 'bookmark', url: 'https://example.com/a', title: 'A page' })
    })

    it('captures a selection with the quote text, a derived title, and the domain tag', () => {
        const draft = draftForMenu(MENU_SELECTION, { url: 'https://example.com/a', title: 'A' }, 'the quote')
        expect(draft).toEqual({ kind: 'selection', url: 'https://example.com/a', title: 'the quote', text: 'the quote', tags: ['example'] })
    })

    it('starts a read-later draft', () => {
        const draft = draftForMenu(MENU_READLATER, { url: 'https://example.com/long', title: 'Long' })
        expect(draft?.kind).toBe('readlater')
    })

    it('starts a summarize draft without page text (grabbed later), tagged with the source domain', () => {
        const draft = draftForMenu(MENU_SUMMARIZE, { url: 'https://turborepo.dev/docs', title: 'Docs' })
        expect(draft).toEqual({ kind: 'summarize', url: 'https://turborepo.dev/docs', title: 'Docs', text: '', tags: ['turborepo'] })
    })

    it('returns null for an unknown menu id', () => {
        expect(draftForMenu('nope', { url: 'https://example.com/a' })).toBeNull()
    })
})

describe('draft storage', () => {
    it('round-trips and clears a draft', async () => {
        const store = memoryStorage()
        const draft: Draft = { kind: 'bookmark', url: 'https://example.com/a', title: 'A' }
        await saveDraft(store, draft)
        expect(await loadDraft(store)).toEqual(draft)
        await clearDraft(store)
        expect(await loadDraft(store)).toBeNull()
    })

    it('treats corrupt draft JSON as absent', async () => {
        const store = memoryStorage({ 'thoth:draft': '{not json' })
        expect(await loadDraft(store)).toBeNull()
    })
})
