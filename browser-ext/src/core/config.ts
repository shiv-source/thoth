import { DEFAULT_BASE_URLS, discoverBaseUrl, probeServer } from './server'
import type { BrowserAPI } from './webext'

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

// LAST_CATEGORY_KEY remembers the last bookmark category the user picked, so a
// new bookmark defaults to it instead of always landing in "unfiled". A
// behavior-based default adapts to how the user files; guessing from the URL
// (which we never read) would misfire as often as it helps.
export const LAST_CATEGORY_KEY = 'thoth:lastCategory'

// loadLastCategory returns the remembered bookmark category, or null when
// there is none yet (first use).
export async function loadLastCategory(store: StorageLike): Promise<string | null> {
    const raw = await store.get(LAST_CATEGORY_KEY)
    return raw || null
}

// saveLastCategory persists the category used for a bookmark save so the next
// capture defaults to it.
export async function saveLastCategory(store: StorageLike, category: string): Promise<void> {
    await store.set(LAST_CATEGORY_KEY, category)
}

export function normalizeBaseUrl(raw: string): string {
    return raw.trim().replace(/\/+$/, '')
}

// isStaticallyCovered reports whether baseUrl's origin is covered by the
// manifests' static host_permissions (http://127.0.0.1:* and
// http://localhost:*). Everything else needs a runtime permission grant for
// the extension's fetch to bypass CORS.
function isStaticallyCovered(baseUrl: string): boolean {
    let u: URL
    try {
        u = new URL(baseUrl)
    } catch {
        return false
    }
    if (u.protocol !== 'http:') return false
    const host = u.hostname.toLowerCase()
    return host === '127.0.0.1' || host === 'localhost'
}

// ensureHostPermission makes sure the extension may reach baseUrl's origin:
// a no-op for the statically permitted localhost hosts, otherwise a
// permissions.request for that origin (the optional_host_permissions grant).
// Returns whether the origin is (or is now) reachable.
export async function ensureHostPermission(api: BrowserAPI, raw: string): Promise<boolean> {
    const base = normalizeBaseUrl(raw)
    if (!base) return true // empty → auto-discovery on localhost, always permitted
    if (isStaticallyCovered(base)) return true
    if (!api.permissions) return false
    const origin = new URL(base).origin
    return api.permissions.request({ origins: [`${origin}/*`] })
}

// resolveBaseUrl returns the server the extension should talk to: a stored
// custom URL when it is reachable, else the first default port that answers,
// else null (the popup's "start thoth" state). Used for background/auto
// connection, where a stale custom URL should fall back to discovery.
export async function resolveBaseUrl(store: StorageLike): Promise<string | null> {
    const custom = normalizeBaseUrl((await store.get(BASE_URL_KEY)) ?? '')
    if (custom && (await probeServer(custom))) return custom
    return discoverBaseUrl(DEFAULT_BASE_URLS)
}

// connectBaseUrl resolves a URL for an explicit user connect: the entered
// custom URL when reachable, else auto-discovery when nothing was entered, else
// null. Unlike resolveBaseUrl it never silently falls back from an unreachable
// custom URL to a default port — the user asked for a specific server.
export async function connectBaseUrl(raw: string): Promise<string | null> {
    const custom = normalizeBaseUrl(raw)
    if (custom) return (await probeServer(custom)) ? custom : null
    return discoverBaseUrl(DEFAULT_BASE_URLS)
}

// saveBaseUrl persists a custom server URL; an empty value clears it and
// returns the extension to auto-discovery.
export async function saveBaseUrl(store: StorageLike, raw: string): Promise<void> {
    const base = normalizeBaseUrl(raw)
    if (base) await store.set(BASE_URL_KEY, base)
    else await store.remove(BASE_URL_KEY)
}
