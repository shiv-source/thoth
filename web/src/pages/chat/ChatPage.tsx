import { useCallback, useEffect, useRef, useState } from 'react'
import { App, Flex } from 'antd'
import { useChat } from '../../hooks/useChat'
import { useConversationRoute } from '../../hooks/useConversationRoute'
import { fetchConversations, fetchSettings, selectConnectionStatus, selectSettings, setStatus } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { ChatSocket } from '../../ws/chat'
import { Composer } from '../../components/chat/Composer'
import { MessageItem } from '../../components/chat/MessageItem'
import { AppHeader } from '../../shared/AppHeader'
import { UsageLine } from '../../components/chat/UsageLine'
import { ChatEmptyState } from '../../components/chat/ChatEmptyState'
import { StatusPill } from '../../components/chat/StatusPill'

function createSocket(): ChatSocket {
    const socket = new ChatSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/v1`)
    socket.connect()
    return socket
}

export function ChatPage({
    onOpenSettings,
    onOpenNote
}: {
    onOpenSettings: () => void
    onOpenNote: (path: string) => void
}) {
    const dispatch = useAppDispatch()
    const [socket, setSocket] = useState<ChatSocket | null>(null)
    // The connection status lives in the store so the whole app can react to
    // it; the socket only reports changes.
    const status = useAppSelector(selectConnectionStatus)
    const {
        messages,
        streaming,
        lastTool,
        thinking,
        thinkingText,
        conversationId,
        lastUsage,
        send,
        cancel,
        load,
        reset
    } = useChat(socket)
    const { message } = App.useApp()
    const endRef = useRef<HTMLDivElement>(null)
    const settings = useAppSelector(selectSettings)
    const model = settings.data?.model ?? ''

    // The composer's model chip reads the default model; settings load on
    // chat mount so it is populated without a settings visit first.
    useEffect(() => {
        void dispatch(fetchSettings())
    }, [dispatch])

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

    // A hidden tab is not "in the chat page": report presence so the server
    // flushes idle pooled CLI processes after its relaxation timeout. Sends on
    // visibility changes and once on mount (a tab can load already hidden).
    useEffect(() => {
        const onVisibility = () => socket?.setPresence(!document.hidden)
        if (document.hidden) socket?.setPresence(false)
        document.addEventListener('visibilitychange', onVisibility)
        return () => document.removeEventListener('visibilitychange', onVisibility)
    }, [socket])

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
                <StatusPill tone="warning">
                    {status === 'reconnecting' ? 'Connection lost — reconnecting…' : 'Connection lost'}
                </StatusPill>
            )}
            {thinking && !lastTool && <StatusPill>Thinking… {thinkingText}</StatusPill>}
            {lastTool && <StatusPill>Reading {lastTool}</StatusPill>}
            {/* Scroll + padding live on a plain div: antd Flex's cssinjs
                reset zeroes both margin AND padding on .ant-flex children,
                so layout utilities must not ride on the Flex element itself.
                Message spacing comes from the Flex's own gap. */}
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                <Flex vertical gap={16}>
                    {messages.length === 0 && <ChatEmptyState onSend={send} />}
                    {messages.map((m, i) => (
                        <MessageItem
                            key={i}
                            message={m}
                            streaming={streaming && i === messages.length - 1}
                            onOpenNote={onOpenNote}
                        />
                    ))}
                    {lastUsage !== null && <UsageLine usage={lastUsage} />}
                    <div ref={endRef} />
                </Flex>
            </div>
            <Composer onSend={send} onCancel={cancel} streaming={streaming} model={model} />
        </div>
    )
}
