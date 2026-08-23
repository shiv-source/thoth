import type { Settings } from '../../api/client'

// settingsBody builds the PUT /api/settings payload from the submitted form
// values and the last-known server state. The whole page is one settings
// object, but each sub-page only renders (and therefore submits) its own
// fields — merge with settings.data so untouched fields always round-trip and
// the backend never rejects a partial body ("wiki_path is required").
export function settingsBody(current: Settings, next: Partial<Settings>): Settings {
    return { ...current, ...next }
}
