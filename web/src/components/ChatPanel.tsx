import { useCallback, useEffect, useRef, useState } from 'react'
import { useChat } from '../hooks/useChat'
import { useConversationRoute } from '../hooks/useConversationRoute'
import { ChatSocket, type ConnectionStatus } from '../ws/chat'
import { Composer } from './Composer'
import { MessageItem } from './MessageItem'
import { SettingsModal } from './SettingsModal'
import { TopBar } from './TopBar'
import { useToast } from './Toast'

function createSocket(): ChatSocket {
  const socket = new ChatSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`)
  socket.connect()
  return socket
}

export function ChatPanel() {
  const [socket, setSocket] = useState<ChatSocket | null>(null)
  const [status, setStatus] = useState<ConnectionStatus>('connected')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const { messages, streaming, lastTool, thinking, thinkingText, conversationId, send, cancel, load, reset } = useChat(socket)
  const { toast } = useToast()
  const endRef = useRef<HTMLDivElement>(null)

  // The socket lives for the panel's lifetime: created on mount (StrictMode
  // remounts close the first one), closed on unmount so no orphan connection
  // or reconnect timer outlives the panel. The retry-once logic fires
  // 'disconnected' exactly once, so the toast can never pile up.
  useEffect(() => {
    const s = createSocket()
    s.onStatusChange((st) => {
      setStatus(st)
      if (st === 'disconnected') toast('Connection lost', 'error')
    })
    setSocket(s)
    return () => s.close()
  }, [toast])

  // /chat/<uuid> deep links and back/forward navigation follow the active
  // conversation; the URL follows conversationId changes.
  const onRouteError = useCallback((message: string) => toast(message, 'error'), [toast])
  useConversationRoute({ socket, conversationId, load, reset, onError: onRouteError })

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streaming])

  const title = messages.find((m) => m.role === 'user')?.content ?? ''
  const displayTitle = title.length === 0
    ? 'New conversation'
    : title.length > 48 ? `${title.slice(0, 48).trimEnd()}…` : title

  return (
    <div className="flex h-full flex-col">
      <TopBar title={displayTitle} onOpenSettings={() => setSettingsOpen(true)} />
      {status !== 'connected' && (
        <div className="flex h-7 shrink-0 items-center gap-2 border-b border-line bg-raised px-4 text-xs text-subtle">
          <span className="h-1.5 w-1.5 rounded-full bg-accent" aria-hidden="true" />
          {status === 'reconnecting' ? 'Connection lost — reconnecting…' : 'Connection lost.'}
        </div>
      )}
      {thinking && !lastTool && (
        <div className="flex h-7 shrink-0 items-center gap-2 border-b border-line bg-raised px-4 text-xs text-subtle">
          <span className="h-1.5 w-1.5 shrink-0 animate-pulse rounded-full bg-accent" aria-hidden="true" />
          <span className="min-w-0 truncate">Thinking… {thinkingText}</span>
        </div>
      )}
      {lastTool && (
        <div className="flex h-7 shrink-0 items-center gap-2 border-b border-line bg-raised px-4 text-xs text-subtle">
          <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-accent" aria-hidden="true" />
          Reading <code className="font-mono text-[11px]">{lastTool}</code>
        </div>
      )}
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-5">
        {messages.length === 0 && (
          <div className="flex h-full flex-col items-center justify-center text-center">
            <div className="font-display text-3xl font-semibold text-heading">Thoth</div>
            <p className="mt-2 max-w-sm text-sm text-subtle">
              Ask anything — “what did we decide in Tuesday's standup?” or
              “save this: the client approved the new roadmap.”
            </p>
          </div>
        )}
        {messages.map((m, i) => (
          <MessageItem key={i} message={m} streaming={streaming && i === messages.length - 1} />
        ))}
        <div ref={endRef} />
      </div>
      <Composer onSend={send} onCancel={cancel} streaming={streaming} />
      {settingsOpen && <SettingsModal onClose={() => setSettingsOpen(false)} />}
    </div>
  )
}
