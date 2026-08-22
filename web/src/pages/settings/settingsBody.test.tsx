import { describe, expect, it } from 'vitest'
import type { Settings } from '../../api/client'
import { settingsBody } from './settingsBody'

// The full server-side settings object a sub-page saves against. Every
// sub-page only submits its own fields, so settingsBody must merge the rest
// back in (the backend rejects a body without wiki_path) while keeping
// provider payloads minimal.
const current: Settings = {
    wiki_path: '~/.thoth/wiki',
    wiki_folders: ['inbox', 'meetings'],
    model: 'deepseek-v4-flash',
    providers: {
        Anthropic: { api_key: '', has_api_key: true, base_url: 'https://api.anthropic.com' },
        OpenAI: { api_key: '', has_api_key: false, base_url: '' }
    },
    repo_url: '',
    sync_enabled: false
}

// Form submit values for the providers page: only the registered base_url /
// api_key fields are present (has_api_key is read-only, never submitted).
type ProviderFormValue = { api_key?: string; base_url: string }
const emptySettings = (providers: Record<string, ProviderFormValue>): Partial<Settings> => ({
    providers: providers as Settings['providers']
})

describe('settingsBody', () => {
    it('merges untouched fields from the current settings so the payload is complete', () => {
        // A Providers-page save carries only provider fields — no wiki_path
        // key at all — so settingsBody must fill the rest in (the backend
        // rejects a body without wiki_path).
        const next = emptySettings({
            OpenAI: { api_key: 'sk-local', base_url: 'http://127.0.0.1:8080' }
        })
        const body = settingsBody(current, next)
        expect(body.wiki_path).toBe('~/.thoth/wiki')
        expect(body.model).toBe('deepseek-v4-flash')
        expect(body.wiki_folders).toEqual(['inbox', 'meetings'])
        expect(body.providers.OpenAI).toEqual({ api_key: 'sk-local', base_url: 'http://127.0.0.1:8080' })
    })

    it('sends only providers whose api_key or base_url changed', () => {
        const next = emptySettings({
            Anthropic: { api_key: '', base_url: 'https://api.anthropic.com' },
            OpenAI: { api_key: 'sk-local', base_url: 'http://127.0.0.1:8080' }
        })
        const body = settingsBody(current, next)
        expect(Object.keys(body.providers)).toEqual(['OpenAI'])
        expect(body.providers.OpenAI).toEqual({ api_key: 'sk-local', base_url: 'http://127.0.0.1:8080' })
    })

    it('drops an untouched provider even when its api_key is empty (write-only)', () => {
        // The server never echoes a stored key, so a seeded provider has
        // api_key '' — matching that must not resubmit (and risk clearing)
        // the provider's state.
        const next = emptySettings({
            Anthropic: { api_key: '', base_url: 'https://api.anthropic.com' }
        })
        expect(settingsBody(current, next).providers).toEqual({})
    })

    it('keeps a provider whose base_url was deliberately cleared', () => {
        const next = emptySettings({
            Anthropic: { api_key: '', base_url: '' }
        })
        expect(settingsBody(current, next).providers).toEqual({ Anthropic: { api_key: '', base_url: '' } })
    })
})
