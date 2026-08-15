import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { useViewShortcuts } from './useViewShortcuts'

function key(init: KeyboardEventInit) {
    act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', init))
    })
}

describe('useViewShortcuts', () => {
    afterEach(() => {
        window.location.hash = ''
    })

    it('navigates views with cmd+1..4 in rail order', () => {
        renderHook(() => useViewShortcuts())
        key({ key: '1', metaKey: true })
        expect(window.location.hash).toBe('#/dashboard')
        key({ key: '2', metaKey: true })
        expect(window.location.hash).toBe('#/chat')
        key({ key: '3', metaKey: true })
        expect(window.location.hash).toBe('#/notes')
        key({ key: '4', metaKey: true })
        expect(window.location.hash).toBe('#/search')
    })

    it('opens search with cmd+K (ctrl on other platforms)', () => {
        renderHook(() => useViewShortcuts())
        key({ key: 'k', metaKey: true })
        expect(window.location.hash).toBe('#/search')
        key({ key: 'k', ctrlKey: true })
        expect(window.location.hash).toBe('#/search')
    })

    it('ignores plain keys and unknown combos', () => {
        renderHook(() => useViewShortcuts())
        key({ key: '2' })
        key({ key: 'x', metaKey: true })
        expect(window.location.hash).toBe('')
    })

    it('removes the listener on unmount', () => {
        const { unmount } = renderHook(() => useViewShortcuts())
        unmount()
        key({ key: '3', metaKey: true })
        expect(window.location.hash).toBe('')
    })
})
