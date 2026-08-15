import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { navigateView, useView, type View } from './useView'

describe('useView', () => {
    afterEach(() => {
        window.location.hash = ''
    })

    it('defaults to chat when the hash is empty', () => {
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('chat')
    })

    it('parses every view from the hash', () => {
        window.location.hash = '#/dashboard'
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('dashboard')
    })

    it('falls back to chat for unknown hashes', () => {
        window.location.hash = '#/nonsense'
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('chat')
    })

    it('follows hash changes (back/forward, navigateView)', () => {
        const { result } = renderHook(() => useView())
        expect(result.current).toBe('chat')

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
})
