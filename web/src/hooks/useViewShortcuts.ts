import { useEffect } from 'react'
import { navigateView, type View } from './useView'

// Mirrors the rail order: dashboard first, then chat, notes, search.
const DIGIT_VIEWS: Record<string, View> = { '1': 'dashboard', '2': 'chat', '3': 'notes', '4': 'search' }

// useViewShortcuts binds ⌘/Ctrl+1..4 to the views and ⌘/Ctrl+K to Search.
// The listener lives on window and is removed on unmount.
export function useViewShortcuts(): void {
    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (!(e.metaKey || e.ctrlKey)) return
            const v = DIGIT_VIEWS[e.key]
            if (v) {
                e.preventDefault()
                navigateView(v)
                return
            }
            if (e.key.toLowerCase() === 'k') {
                e.preventDefault()
                navigateView('search')
            }
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    }, [])
}
