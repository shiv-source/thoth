import { useEffect, useState } from 'react'

export type View = 'chat' | 'notes' | 'dashboard' | 'search' | 'settings'

export interface ViewRoute {
    view: View
    // segment is the decoded path part after the view: the open note for
    // /notes/<path>, the tab for /settings/<tab>. It rides the URL, so the
    // state survives reloads and back/forward.
    segment: string | null
    // query is the URL's ?q= parameter — the search view's state.
    query: string | null
}

// Pathname routing: / is the dashboard (home), /chat carries the
// conversation id (owned by useConversationRoute — the uuid segment is not
// part of the view route), and the other views carry their state as the
// path segment (or, for search, the ?q= query parameter).
const VIEW_PATH = /^\/(chat|notes|dashboard|search|settings)(?:\/(.+))?$/

function routeFromPathname(pathname: string): ViewRoute {
    const m = VIEW_PATH.exec(pathname)
    if (!m) return { view: 'dashboard', segment: null, query: null }
    const view = m[1] as View
    const segment = (view === 'notes' || view === 'settings') && m[2] !== undefined ? decodeURIComponent(m[2]) : null
    const query = view === 'search' ? new URLSearchParams(window.location.search).get('q') : null
    return { view, segment, query }
}

// setPath pushes the URL the way a user link would, so the route hooks'
// applyRoute runs — pushState alone does not fire popstate.
function setPath(path: string): void {
    window.history.pushState(null, '', path)
    window.dispatchEvent(new PopStateEvent('popstate'))
}

// navigateView moves the app between views (dropping any segment).
// The dashboard is the root path.
export function navigateView(v: View): void {
    setPath(v === 'dashboard' ? '/' : `/${v}`)
}

// navigateSegment moves to a view carrying state in the URL segment.
// Slashes stay readable; decodeURIComponent reverses it on parse.
export function navigateSegment(v: View, segment: string | null): void {
    if (segment === null) {
        navigateView(v)
        return
    }
    const encoded = encodeURIComponent(segment).replace(/%2F/gi, '/')
    setPath(`/${v}/${encoded}`)
}

// navigateNote opens (or clears) the note in the Notes view and switches to
// it. The path rides the URL, so the open note survives a reload.
export function navigateNote(path: string | null): void {
    navigateSegment('notes', path)
}

// useViewRoute maps the URL pathname to the active view plus its segment.
// The dashboard is the home view for any path the app does not recognize.
export function useViewRoute(): ViewRoute {
    const [route, setRoute] = useState<ViewRoute>(() => routeFromPathname(window.location.pathname))
    useEffect(() => {
        const applyRoute = () => setRoute(routeFromPathname(window.location.pathname))
        window.addEventListener('popstate', applyRoute)
        return () => window.removeEventListener('popstate', applyRoute)
    }, [])
    return route
}

// useView is the view-only slice of useViewRoute.
export function useView(): View {
    return useViewRoute().view
}
