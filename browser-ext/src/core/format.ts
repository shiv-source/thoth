import type { CaptureInput, Draft } from './types'

export const DEFAULT_CATEGORY = 'unfiled'
export const BOOKMARK_CATEGORIES = ['reference', 'article', 'docs', 'reading', 'ideas', 'unfiled'] as const

// MAX_CAPTURE_TEXT bounds a captured page/selection body in characters. The
// server rejects larger bodies (maxCaptureText), so the extension truncates
// before sending — a summarization never needs more than this.
export const MAX_CAPTURE_TEXT = 500_000

// MAX_TITLE is the cap for derived capture titles (mirrors the store's 60-rune
// title cap); a longer first sentence is cut at the word boundary.
const MAX_TITLE = 60

// selectionTitle derives a note title from a captured selection: the first
// sentence-bearing line (real content, not a nav label or heading fragment),
// stripped of markdown heading/blockquote markers, capped at MAX_TITLE. A
// selection inheriting the page's generic title ("Introduction") is far less
// identifiable than its own first sentence.
export function selectionTitle(text: string): string {
    const cleaned = text
        .split(/\r?\n/)
        .map((line) => line.replace(/^#{1,6}\s+/, '').replace(/^>\s?/, '').trim())
        .filter((line) => line.length > 0)
    if (cleaned.length === 0) return ''
    const pick = cleaned.find((line) => /[.!?]/.test(line)) ?? cleaned[0]!
    const end = pick.search(/[.!?](?=\s|$)/)
    const title = end >= 0 ? pick.slice(0, end + 1) : pick
    return title.length > MAX_TITLE ? title.slice(0, MAX_TITLE).trimEnd() : title
}

// sourceTag derives a consistent capture tag from a source URL's bare host
// (https://turborepo.dev/docs → "turborepo"), so every capture from the same
// site stays grouped and searchable. Returns undefined when the URL has no
// usable host.
const COMMON_TLDS = new Set([
    'com', 'org', 'net', 'io', 'dev', 'co', 'uk', 'app', 'site', 'info', 'xyz', 'tv', 'me', 'us',
    'ca', 'au', 'in', 'de', 'fr', 'jp', 'ru', 'github', 'gitlab', 'md', 'ai', 'so', 'tech',
    'online', 'blog', 'shop', 'store', 'cloud'
])

export function sourceTag(sourceUrl: string): string | undefined {
    try {
        const host = new URL(sourceUrl).hostname.toLowerCase().replace(/^www\./, '')
        const labels = host.split('.').filter(Boolean)
        while (labels.length > 1 && COMMON_TLDS.has(labels[labels.length - 1]!)) labels.pop()
        const tag = labels[labels.length - 1]
        return tag && tag.length > 0 ? tag : undefined
    } catch {
        return undefined
    }
}

// The scaffold folders the popup offers before the server's settings arrive.
export const DEFAULT_NOTE_FOLDERS = [
    'inbox',
    'meetings',
    'projects',
    'links',
    'setup',
    'knowledge',
    'todos',
    'daily',
] as const

// selectionToBody renders a captured selection as the note body: the quote in
// a blockquote with a source attribution link, per the rulebook's capture
// convention. Returns '' for an empty quote.
export function selectionToBody(quote: string, sourceUrl: string, pageTitle: string): string {
    const trimmed = quote.trim()
    if (!trimmed) return ''
    const quoted = trimmed
        .split(/\r?\n/)
        .map((line) => `> ${line.trim()}`)
        .join('\n')
    const source = pageTitle.trim() || sourceUrl
    return `${quoted}\n\n> — [${source}](${sourceUrl})`
}

// isHttpUrl reports whether u is an absolute http(s) URL — the only scheme the
// wiki accepts for capture provenance.
export function isHttpUrl(u: string): boolean {
    try {
        const parsed = new URL(u)
        return parsed.protocol === 'http:' || parsed.protocol === 'https:'
    } catch {
        return false
    }
}

// sanitizeSingleLine collapses whitespace so a bookmark reason or category
// stays one line (the server rejects newlines there).
export function sanitizeSingleLine(s: string): string {
    return s.replace(/\s+/g, ' ').trim()
}

// parseTags splits a comma-separated tag input into trimmed, non-empty tags.
export function parseTags(raw: string): string[] {
    return raw
        .split(',')
        .map((tag) => tag.trim())
        .filter((tag) => tag.length > 0)
}

// draftToCapture maps an edited draft to the wire body for its kind, dropping
// fields the server ignores per kind. Selection is folded into note (the
// quote already lives in text). Summarize is never sent through capture — it
// rides POST /api/v1/capture/summarize — so its case is unreachable.
export function draftToCapture(draft: Draft): CaptureInput {
    const base = {
        url: draft.url.trim() || undefined,
        title: draft.title.trim() || undefined,
        text: draft.text?.trim() || undefined,
    }
    switch (draft.kind) {
        case 'bookmark':
            return {
                kind: 'bookmark',
                ...base,
                category: (draft.category ?? DEFAULT_CATEGORY).trim(),
                reason: draft.reason?.trim() || undefined,
            }
        case 'selection':
        case 'note':
            return {
                kind: 'note',
                ...base,
                folder: draft.folder?.trim() || 'inbox',
                tags: draft.tags ?? [],
            }
        case 'readlater':
            return {
                kind: 'readlater',
                ...base,
                reason: draft.reason?.trim() || undefined,
            }
        case 'summarize':
            return { kind: 'note', ...base, folder: 'knowledge' }
    }
}

// openNoteUrl builds the dashboard URL that opens a saved note, mirroring the
// dashboard's pathname routing (/notes/<path>, slashes readable).
export function openNoteUrl(baseUrl: string, path: string): string {
    return `${baseUrl}/notes/${encodeURIComponent(path).replace(/%2F/gi, '/')}`
}
