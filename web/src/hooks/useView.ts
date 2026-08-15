import { useEffect, useState } from 'react'

export type View = 'chat' | 'notes' | 'dashboard' | 'search' | 'settings'

const VIEW_HASH = /^#\/(chat|notes|dashboard|search|settings)/

function viewFromHash(hash: string): View {
    const m = VIEW_HASH.exec(hash)
    return (m?.[1] as View | undefined) ?? 'chat'
}

// navigateView moves the app between views. hashchange drives useView, but
// assigning the hash alone does not fire it in every environment (jsdom),
// so dispatch it explicitly — same pattern as the conversation navigate.
export function navigateView(v: View): void {
    window.location.hash = `#/${v}`
    window.dispatchEvent(new HashChangeEvent('hashchange'))
}

// useView maps the URL hash to the active view. Chat is the home view for
// any hash the app does not recognize. Back/forward work via hashchange.
export function useView(): View {
    const [view, setView] = useState<View>(() => viewFromHash(window.location.hash))
    useEffect(() => {
        const onHash = () => setView(viewFromHash(window.location.hash))
        window.addEventListener('hashchange', onHash)
        return () => window.removeEventListener('hashchange', onHash)
    }, [])
    return view
}
