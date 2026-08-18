import { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, App } from 'antd'
import { useChat } from '../hooks/useChat'
import { useConversationRoute } from '../hooks/useConversationRoute'
import { fetchConversations, selectConnectionStatus, setStatus } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { ChatSocket } from '../ws/chat'
import { Composer } from './Composer'
import { MessageItem } from './MessageItem'
import { AppHeader } from './AppHeader'

function createSocket(): ChatSocket {
    const socket = new ChatSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`)
    socket.connect()
    return socket
}

export function ChatPanel({ onOpenSettings }: { onOpenSettings: () => void }) {
    const dispatch = useAppDispatch()
    const [socket, setSocket] = useState<ChatSocket | null>(null)
    // The connection status lives in the store so the whole app can react to
    // it; the socket only reports changes.
    const status = useAppSelector(selectConnectionStatus)
    const { messages, streaming, lastTool, thinking, thinkingText, conversationId, send, cancel, load, reset } =
        useChat(socket)
    const { message } = App.useApp()
    const endRef = useRef<HTMLDivElement>(null)

    // The socket lives for the panel's lifetime: created on mount (StrictMode
    // remounts close the first one), closed on unmount so no orphan connection
    // or reconnect timer outlives the panel. The retry-once logic fires
    // 'disconnected' exactly once, so the toast can never pile up.
    useEffect(() => {
        const s = createSocket()
        s.onStatusChange((st) => {
            dispatch(setStatus(st))
            if (st === 'disconnected') void message.error('Connection lost')
        })
        setSocket(s)
        return () => s.close()
    }, [message, dispatch])

    // /chat/<uuid> deep links and back/forward navigation follow the active
    // conversation; the URL follows conversationId changes.
    const onRouteError = useCallback((err: string) => void message.error(err), [message])
    useConversationRoute({ socket, conversationId, load, reset, onError: onRouteError })

    const prevConversationId = useRef<string | null>(null)

    // The URL push on turn_done is silent (no popstate), so the sidebar would
    // not refetch and a freshly created chat stays hidden until the next
    // navigation — refresh the list here, once per new conversation.
    useEffect(() => {
        if (conversationId !== null && prevConversationId.current === null) {
            void dispatch(fetchConversations())
        }
        prevConversationId.current = conversationId
    }, [conversationId, dispatch])

    useEffect(() => {
        endRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [messages, streaming])

    const title = messages.find((m) => m.role === 'user')?.content ?? ''
    const displayTitle =
        title.length === 0 ? 'New conversation' : title.length > 48 ? `${title.slice(0, 48).trimEnd()}…` : title

    return (
        <div className="flex h-full flex-col">
            <AppHeader title={displayTitle} onOpenSettings={onOpenSettings} />
            {status !== 'connected' && (
                <Alert
                    type="warning"
                    banner
                    message={status === 'reconnecting' ? 'Connection lost — reconnecting…' : 'Connection lost.'}
                />
            )}
            {thinking && !lastTool && (
                <Alert type="info" banner message={<span className="truncate">Thinking… {thinkingText}</span>} />
            )}
            {lastTool && (
                <Alert
                    type="info"
                    banner
                    message={
                        <>
                            Reading <code className="font-mono text-[11px]">{lastTool}</code>
                        </>
                    }
                />
            )}
            <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-5">
                {messages.length === 0 && (
                    <div className="flex h-full flex-col items-center justify-center text-center">
                        <div className="font-display text-3xl font-semibold text-heading">Thoth</div>
                        <p className="mt-2 max-w-sm text-sm text-subtle">
                            Ask anything — “what did we decide in Tuesday's standup?” or “save this: the client approved
                            the new roadmap.”
                        </p>
                    </div>
                )}
                {messages.map((m, i) => (
                    <MessageItem key={i} message={m} streaming={streaming && i === messages.length - 1} />
                ))}
                <div ref={endRef} />
            </div>
            <Composer onSend={send} onCancel={cancel} streaming={streaming} />
        </div>
    )
}
