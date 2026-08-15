import { useEffect, useState } from 'react'

export type View = 'chat' | 'notes' | 'dashboard' | 'search' | 'settings'

export interface ViewRoute {
    view: View
    // note is the note path carried by #/notes/<path> — the open note in
    // the Notes view survives reloads and back/forward.
    note: string | null
}

const VIEW_HASH = /^#\/(chat|notes|dashboard|search|settings)(?:\/(.+))?$/

function routeFromHash(hash: string): ViewRoute {
    const m = VIEW_HASH.exec(hash)
    if (!m) return { view: 'chat', note: null }
    const view = m[1] as View
    const note = view === 'notes' && m[2] !== undefined ? decodeURIComponent(m[2]) : null
    return { view, note }
}

// navigateView moves the app between views; hashchange drives useView.
// Assigning the hash alone does not fire hashchange in every environment
// (jsdom), so dispatch it explicitly — same pattern as the conversation
// navigate.
export function navigateView(v: View): void {
    window.location.hash = `#/${v}`
    window.dispatchEvent(new HashChangeEvent('hashchange'))
}

// navigateNote opens (or clears) the note in the Notes view and switches to
// it. The path rides the URL, so the open note survives a reload.
export function navigateNote(path: string | null): void {
    // Slashes stay readable in the hash; decodeURIComponent reverses it.
    const encoded = path ? encodeURIComponent(path).replace(/%2F/gi, '/') : ''
    window.location.hash = path ? `#/notes/${encoded}` : '#/notes'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
}

// useViewRoute maps the URL hash to the active view plus the open note.
// Chat is the home view for any hash the app does not recognize.
export function useViewRoute(): ViewRoute {
    const [route, setRoute] = useState<ViewRoute>(() => routeFromHash(window.location.hash))
    useEffect(() => {
        const onHash = () => setRoute(routeFromHash(window.location.hash))
        window.addEventListener('hashchange', onHash)
        return () => window.removeEventListener('hashchange', onHash)
    }, [])
    return route
}

// useView is the view-only slice of useViewRoute.
export function useView(): View {
    return useViewRoute().view
}
