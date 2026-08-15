import { useEffect, useState } from 'react'

export type View = 'chat' | 'notes' | 'dashboard' | 'search' | 'settings'

export interface ViewRoute {
    view: View
    // segment is the decoded path part after the view: the open note for
    // #/notes/<path>, the query for #/search/<q>, the tab for
    // #/settings/<tab>. It rides the URL, so the state survives reloads
    // and back/forward.
    segment: string | null
}

const VIEW_HASH = /^#\/(chat|notes|dashboard|search|settings)(?:\/(.+))?$/

// A /chat/<uuid> deep link with no view hash means the user came for that
// conversation — land on chat. Otherwise the Dashboard is the home view.
const CHAT_PATH = /^\/chat\/[0-9a-fA-F-]{36}$/

function routeFromHash(hash: string): ViewRoute {
    const m = VIEW_HASH.exec(hash)
    if (!m) {
        return { view: CHAT_PATH.test(window.location.pathname) ? 'chat' : 'dashboard', segment: null }
    }
    return {
        view: m[1] as View,
        segment: m[2] !== undefined ? decodeURIComponent(m[2]) : null
    }
}

// setHash assigns the hash and dispatches hashchange explicitly — assigning
// alone does not fire it in every environment (jsdom), the same pattern as
// the conversation navigate.
function setHash(hash: string): void {
    window.location.hash = hash
    window.dispatchEvent(new HashChangeEvent('hashchange'))
}

// navigateView moves the app between views (dropping any segment).
export function navigateView(v: View): void {
    setHash(`#/${v}`)
}

// navigateSegment moves to a view carrying state in the URL segment.
// Slashes stay readable; decodeURIComponent reverses it on parse.
export function navigateSegment(v: View, segment: string | null): void {
    if (segment === null) {
        setHash(`#/${v}`)
        return
    }
    const encoded = encodeURIComponent(segment).replace(/%2F/gi, '/')
    setHash(`#/${v}/${encoded}`)
}

// navigateNote opens (or clears) the note in the Notes view and switches to
// it. The path rides the URL, so the open note survives a reload.
export function navigateNote(path: string | null): void {
    navigateSegment('notes', path)
}

// useViewRoute maps the URL hash to the active view plus its segment. Chat
// is the home view for any hash the app does not recognize.
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
