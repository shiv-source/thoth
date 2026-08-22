import type { Settings } from '../../api/client'

// settingsBody builds the PUT /api/settings payload from the submitted form
// values and the last-known server state. The whole page is one settings
// object, but each sub-page only renders (and therefore submits) its own
// fields — merge with settings.data so untouched fields always round-trip
// and the backend never rejects a partial body ("wiki_path is required").
//
// Providers are only included when the user actually changed them: api_key
// is write-only (the server never echoes it, so a seeded value is always
// ""), and an untouched base_url must never resubmit an empty value that
// would clear a stored override.
export function settingsBody(current: Settings, next: Partial<Settings>): Settings {
    const providers = Object.fromEntries(
        Object.entries(next.providers ?? {}).filter(([name, pc]) => {
            const was = current.providers?.[name]
            return pc.api_key !== (was?.api_key ?? '') || pc.base_url !== (was?.base_url ?? '')
        })
    )
    return { ...current, ...next, providers }
}
