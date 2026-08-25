import type { StorageLike } from './config'
import type { Draft } from './types'

export const DRAFT_KEY = 'thoth:draft'

// loadDraft reads the pending capture draft (written by a context-menu click
// or the capture command), or null when there is none or it is unreadable.
export async function loadDraft(store: StorageLike): Promise<Draft | null> {
    const raw = await store.get(DRAFT_KEY)
    if (!raw) return null
    try {
        return JSON.parse(raw) as Draft
    } catch {
        return null
    }
}

export async function saveDraft(store: StorageLike, draft: Draft): Promise<void> {
    await store.set(DRAFT_KEY, JSON.stringify(draft))
}

export async function clearDraft(store: StorageLike): Promise<void> {
    await store.remove(DRAFT_KEY)
}
