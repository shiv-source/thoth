import { describe, expect, it } from 'vitest'
import {
    DEFAULT_CATEGORY,
    draftToCapture,
    isHttpUrl,
    openNoteUrl,
    parseTags,
    sanitizeSingleLine,
    selectionToBody,
} from '../src/core/format'
import type { Draft } from '../src/core/types'

describe('selectionToBody', () => {
    it('renders the quote in a blockquote with a source attribution', () => {
        const body = selectionToBody('line one\nline two', 'https://example.com/a', 'A page')
        expect(body).toBe('> line one\n> line two\n\n> — [A page](https://example.com/a)')
    })

    it('returns empty for an empty quote', () => {
        expect(selectionToBody('  \n ', 'https://example.com/a', 'A')).toBe('')
    })
})

describe('isHttpUrl', () => {
    it('accepts http and https absolute URLs', () => {
        expect(isHttpUrl('https://example.com/x')).toBe(true)
        expect(isHttpUrl('http://127.0.0.1:8333')).toBe(true)
    })
    it('rejects other schemes and non-URLs', () => {
        expect(isHttpUrl('ftp://example.com/x')).toBe(false)
        expect(isHttpUrl('javascript:alert(1)')).toBe(false)
        expect(isHttpUrl('not a url')).toBe(false)
        expect(isHttpUrl('')).toBe(false)
    })
})

describe('sanitizeSingleLine', () => {
    it('collapses whitespace to a single line', () => {
        expect(sanitizeSingleLine('  docs\n  reference ')).toBe('docs reference')
    })
})

describe('parseTags', () => {
    it('splits commas, trims, and drops empties', () => {
        expect(parseTags('  go , channels ,, ')).toEqual(['go', 'channels'])
    })
    it('returns [] for empty input', () => {
        expect(parseTags('')).toEqual([])
    })
})

describe('draftToCapture', () => {
    it('maps a bookmark draft to the wire body', () => {
        const draft: Draft = {
            kind: 'bookmark',
            url: 'https://example.com/a',
            title: 'A',
            category: 'reference',
            reason: 'docs',
        }
        expect(draftToCapture(draft)).toEqual({
            kind: 'bookmark',
            url: 'https://example.com/a',
            title: 'A',
            category: 'reference',
            reason: 'docs',
        })
    })

    it('defaults bookmark category to unfiled', () => {
        const draft: Draft = { kind: 'bookmark', url: 'https://example.com/a', title: 'A' }
        expect(draftToCapture(draft).category).toBe(DEFAULT_CATEGORY)
    })

    it('maps a selection to a note with the quote as text', () => {
        const draft: Draft = {
            kind: 'selection',
            url: 'https://example.com/a',
            title: 'A',
            text: '> quote',
            folder: 'knowledge',
            tags: ['q'],
        }
        expect(draftToCapture(draft)).toEqual({
            kind: 'note',
            url: 'https://example.com/a',
            title: 'A',
            text: '> quote',
            folder: 'knowledge',
            tags: ['q'],
        })
    })

    it('maps a quick note to inbox by default', () => {
        const draft: Draft = { kind: 'note', url: '', title: '', text: 'thought' }
        expect(draftToCapture(draft)).toEqual({ kind: 'note', text: 'thought', folder: 'inbox', tags: [] })
    })

    it('maps a read-later draft flat', () => {
        const draft: Draft = { kind: 'readlater', url: 'https://example.com/long', title: 'Long', reason: 'read' }
        expect(draftToCapture(draft)).toEqual({
            kind: 'readlater',
            url: 'https://example.com/long',
            title: 'Long',
            reason: 'read',
        })
    })
})

describe('openNoteUrl', () => {
    it('builds the dashboard note URL with readable slashes', () => {
        expect(openNoteUrl('http://127.0.0.1:8333', 'knowledge/summary.md')).toBe(
            'http://127.0.0.1:8333/notes/knowledge/summary.md'
        )
    })
})
