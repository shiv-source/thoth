import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { navigateNote, navigateSegment, navigateView, useView, useViewRoute, type View } from './useView'

describe('useView', () => {
    afterEach(() => {
        window.location.hash = ''
    })

    it('defaults to dashboard when the hash is empty', () => {
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('lands on chat for a /chat/<uuid> deep link with no hash', () => {
        window.history.pushState(null, '', '/chat/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('chat')
        window.history.pushState(null, '', '/')
    })

    it('parses every view from the hash', () => {
        window.location.hash = '#/dashboard'
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('falls back to dashboard for unknown hashes', () => {
        window.location.hash = '#/nonsense'
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('rejects near-miss hashes (prefix of a view name)', () => {
        window.location.hash = '#/notesfoo'
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('follows hash changes (back/forward, navigateView)', () => {
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')

        act(() => {
            navigateView('notes')
        })
        expect(result.current).toBe('notes')

        act(() => {
            window.location.hash = '#/chat'
            window.dispatchEvent(new HashChangeEvent('hashchange'))
        })
        expect(result.current).toBe('chat')
    })

    it('navigateView writes the hash for every view', () => {
        const views: View[] = ['chat', 'notes', 'dashboard', 'search', 'settings']
        for (const v of views) {
            act(() => navigateView(v))
            expect(window.location.hash).toBe(`#/${v}`)
        }
    })

    it('restores the open note from #/notes/<path> on load', () => {
        window.location.hash = '#/notes/knowledge/renovate-github-action.md'
        const { result } = renderHook(() => useViewRoute())
        expect(result.current.view).toBe('notes')
        expect(result.current.segment).toBe('knowledge/renovate-github-action.md')
    })

    it('treats #/notes without a path as no open note', () => {
        window.location.hash = '#/notes'
        const { result } = renderHook(() => useViewRoute())
        expect(result.current.view).toBe('notes')
        expect(result.current.segment).toBeNull()
    })

    it('navigateNote opens and clears the note through the hash', () => {
        renderHook(() => useViewRoute())
        act(() => navigateNote('meetings/standup.md'))
        expect(window.location.hash).toBe('#/notes/meetings/standup.md')
        act(() => navigateNote(null))
        expect(window.location.hash).toBe('#/notes')
    })

    it('carries the search query and settings tab as the segment', () => {
        window.location.hash = '#/search/bookmarks'
        const search = renderHook(() => useViewRoute())
        expect(search.result.current).toEqual({ view: 'search', segment: 'bookmarks' })
        search.unmount()

        window.location.hash = '#/settings/doctor'
        const settings = renderHook(() => useViewRoute())
        expect(settings.result.current).toEqual({ view: 'settings', segment: 'doctor' })
        settings.unmount()
    })

    it('navigateSegment writes the segment for any view', () => {
        renderHook(() => useViewRoute())
        act(() => navigateSegment('settings', 'git'))
        expect(window.location.hash).toBe('#/settings/git')
    })

    it('switching views drops the note segment', () => {
        renderHook(() => useViewRoute())
        act(() => navigateNote('a.md'))
        act(() => navigateView('dashboard'))
        expect(window.location.hash).toBe('#/dashboard')
    })
})
