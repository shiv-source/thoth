import { describe, expect, it } from 'vitest'
import type { Settings } from '../../api/client'
import { settingsBody } from './settingsBody'

// The full server-side settings object a sub-page saves against. Every
// sub-page only submits its own fields, so settingsBody must merge the rest
// back in (the backend rejects a body without wiki_path).
const current: Settings = {
    wiki_path: '~/.thoth/wiki',
    wiki_folders: ['inbox', 'meetings'],
    model: 'deepseek-v4-flash',
    context_injection: false,
    conversation_retention_days: 7
}

describe('settingsBody', () => {
    it('merges untouched fields from the current settings so the payload is complete', () => {
        // A Providers-page save no longer carries provider credentials (they
        // live on the providers table now), so a partial body must still
        // round-trip the rest of the settings object.
        const body = settingsBody(current, { context_injection: true })
        expect(body.wiki_path).toBe('~/.thoth/wiki')
        expect(body.model).toBe('deepseek-v4-flash')
        expect(body.wiki_folders).toEqual(['inbox', 'meetings'])
        expect(body.context_injection).toBe(true)
    })

    it('carries every provided field into the merged body', () => {
        const body = settingsBody(current, { model: 'claude-sonnet-5' })
        expect(body.model).toBe('claude-sonnet-5')
        expect(body.wiki_path).toBe('~/.thoth/wiki')
    })
})
