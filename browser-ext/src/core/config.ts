import { DEFAULT_BASE_URLS, discoverBaseUrl, probeServer } from './server'

// StorageLike is the minimal KV surface both the background worker and the
// popup need — the real one is ext.storage.local (see webext.ts); tests inject
// an in-memory fake.
export interface StorageLike {
    get(key: string): Promise<string | undefined>
    set(key: string, value: string): Promise<void>
    remove(key: string): Promise<void>
}

// BASE_URL_KEY holds the user's custom server URL; when absent the extension
// discovers across the default ports (8333 then 8334).
export const BASE_URL_KEY = 'thoth:baseUrl'

export function normalizeBaseUrl(raw: string): string {
    return raw.trim().replace(/\/+$/, '')
}

// resolveBaseUrl returns the server the extension should talk to: a stored
// custom URL when it is reachable, else the first default port that answers,
// else null (the popup's "start thoth" state).
export async function resolveBaseUrl(store: StorageLike): Promise<string | null> {
    const custom = normalizeBaseUrl((await store.get(BASE_URL_KEY)) ?? '')
    if (custom && (await probeServer(custom))) return custom
    return discoverBaseUrl(DEFAULT_BASE_URLS)
}

// saveBaseUrl persists a custom server URL; an empty value clears it and
// returns the extension to auto-discovery.
export async function saveBaseUrl(store: StorageLike, raw: string): Promise<void> {
    const base = normalizeBaseUrl(raw)
    if (base) await store.set(BASE_URL_KEY, base)
    else await store.remove(BASE_URL_KEY)
}
