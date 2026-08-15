import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { navigateNote, navigateSegment, navigateView, useView, useViewRoute, type View } from './useView'

function go(path: string) {
    window.history.pushState(null, '', path)
}

describe('useView', () => {
    afterEach(() => {
        go('/')
    })

    it('defaults to dashboard on the root path', () => {
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('parses every view from the pathname', () => {
        go('/notes')
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('notes')
    })

    it('falls back to dashboard for unknown paths', () => {
        go('/nonsense')
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('treats /chat/<uuid> as the chat view', () => {
        go('/chat/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('chat')
    })

    it('follows popstate (back/forward, navigateView)', () => {
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')

        act(() => {
            navigateView('notes')
        })
        expect(result.current).toBe('notes')

        act(() => {
            go('/search')
            window.dispatchEvent(new PopStateEvent('popstate'))
        })
        expect(result.current).toBe('search')
    })

    it('navigateView writes the path for every view', () => {
        const views: View[] = ['chat', 'notes', 'dashboard', 'search', 'settings']
        for (const v of views) {
            act(() => navigateView(v))
            expect(window.location.pathname).toBe(v === 'dashboard' ? '/' : `/${v}`)
        }
    })

    it('restores the open note from /notes/<path> on load', () => {
        go('/notes/knowledge/renovate-github-action.md')
        const { result } = renderHook(() => useViewRoute())
        expect(result.current.view).toBe('notes')
        expect(result.current.segment).toBe('knowledge/renovate-github-action.md')
    })

    it('treats /notes without a path as no open note', () => {
        go('/notes')
        const { result } = renderHook(() => useViewRoute())
        expect(result.current.view).toBe('notes')
        expect(result.current.segment).toBeNull()
    })

    it('navigateNote opens and clears the note through the path', () => {
        renderHook(() => useViewRoute())
        act(() => navigateNote('meetings/standup.md'))
        expect(window.location.pathname).toBe('/notes/meetings/standup.md')
        act(() => navigateNote(null))
        expect(window.location.pathname).toBe('/notes')
    })

    it('carries the search query and settings tab as the segment', () => {
        go('/search/bookmarks')
        const search = renderHook(() => useViewRoute())
        expect(search.result.current).toEqual({ view: 'search', segment: 'bookmarks' })
        search.unmount()

        go('/settings/doctor')
        const settings = renderHook(() => useViewRoute())
        expect(settings.result.current).toEqual({ view: 'settings', segment: 'doctor' })
        settings.unmount()
    })

    it('navigateSegment writes the segment for any view', () => {
        renderHook(() => useViewRoute())
        act(() => navigateSegment('settings', 'git'))
        expect(window.location.pathname).toBe('/settings/git')
    })

    it('switching views drops the segment', () => {
        renderHook(() => useViewRoute())
        act(() => navigateNote('a.md'))
        act(() => navigateView('dashboard'))
        expect(window.location.pathname).toBe('/')
    })
})
