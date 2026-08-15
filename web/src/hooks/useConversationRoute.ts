import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { ChatSocket } from '../ws/chat'
import type { ChatMessage } from './useChat'

// /chat/<uuid> — the conversation id is a 36-char dashed UUID.
const CHAT_PATH = /^\/chat\/([0-9a-fA-F-]{36})$/

function chatIdFromPath(pathname: string): string | null {
    const m = CHAT_PATH.exec(pathname)
    return m ? (m[1] ?? null) : null
}

// navigate changes the URL the way a user link would, so the route hook's
// applyRoute handles loading/pinning/reset — pushState alone does not fire
// popstate, so dispatch it explicitly.
export function navigate(path: string): void {
    window.history.pushState(null, '', path)
    window.dispatchEvent(new PopStateEvent('popstate'))
}

export interface ConversationRouteOptions {
    socket: ChatSocket | null
    conversationId: string | null
    load: (msgs: ChatMessage[], convId: string) => void
    reset: () => void
    onError: (message: string) => void
}

// useConversationRoute keeps the URL and the active conversation in sync.
// State → URL: once hydrated, a conversationId change (turn_done, history
// pick, new chat) follows the path via pushState. URL → state: on mount and
// on popstate (back/forward), /chat/<uuid> loads the conversation through
// the same fetch+load+open flow the history menu uses; unknown ids fall
// back to a fresh chat. The state→URL effect stays silent while a route
// fetch is in flight so it cannot clobber a back/forward navigation.
export function useConversationRoute(opts: ConversationRouteOptions): void {
    const { socket, conversationId, load, reset, onError } = opts
    const conversationIdRef = useRef(conversationId)
    conversationIdRef.current = conversationId
    const [hydrated, setHydrated] = useState(false)

    useEffect(() => {
        const applyRoute = () => {
            const id = chatIdFromPath(window.location.pathname)
            if (id === null) {
                setHydrated(true)
                if (conversationIdRef.current !== null) reset()
                return
            }
            if (id === conversationIdRef.current) {
                // Same conversation (e.g. the socket arrived after the fetch, or a
                // popstate loop) — re-pin the server side, never refetch.
                socket?.open(id)
                setHydrated(true)
                return
            }
            setHydrated(false) // suppress URL sync until this route settles
            api.getConversation(id)
                .then((res) => {
                    load(
                        res.messages.map((m) => ({ role: m.role, content: m.content })),
                        id
                    )
                    socket?.open(id)
                })
                .catch(() => {
                    onError('Conversation not found')
                    window.history.replaceState(null, '', '/')
                    reset()
                })
                .finally(() => setHydrated(true))
        }
        applyRoute()
        window.addEventListener('popstate', applyRoute)
        return () => window.removeEventListener('popstate', applyRoute)
    }, [socket, load, reset, onError])

    useEffect(() => {
        if (!hydrated) return
        const want = conversationId ? `/chat/${conversationId}` : '/chat'
        if (window.location.pathname !== want) {
            window.history.pushState(null, '', want)
        }
    }, [hydrated, conversationId])
}
