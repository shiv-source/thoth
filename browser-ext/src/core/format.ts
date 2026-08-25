import type { CaptureInput, Draft } from './types'

export const DEFAULT_CATEGORY = 'unfiled'
export const BOOKMARK_CATEGORIES = ['reference', 'article', 'docs', 'reading', 'ideas', 'unfiled'] as const

// MAX_CAPTURE_TEXT bounds a captured page/selection body in characters. The
// server rejects larger bodies (maxCaptureText), so the extension truncates
// before sending — a summarization never needs more than this.
export const MAX_CAPTURE_TEXT = 500_000

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
