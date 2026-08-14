import { useEffect, useRef, useState } from 'react'
import { useChat } from '../hooks/useChat'
import { ChatSocket, type ConnectionStatus } from '../ws/chat'
import { Composer } from './Composer'
import { MessageItem } from './MessageItem'

function createSocket(): ChatSocket {
  const socket = new ChatSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`)
  socket.connect()
  return socket
}

export function ChatPanel() {
  const [socket, setSocket] = useState<ChatSocket | null>(null)
  const [status, setStatus] = useState<ConnectionStatus>('connected')
  const { messages, streaming, send, cancel } = useChat(socket)
  const endRef = useRef<HTMLDivElement>(null)

  // The socket lives for the panel's lifetime: created on mount (StrictMode
  // remounts close the first one), closed on unmount so no orphan connection
  // or reconnect timer outlives the panel.
  useEffect(() => {
    const s = createSocket()
    s.onStatusChange(setStatus)
    setSocket(s)
    return () => s.close()
  }, [])

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streaming])

  return (
    <div className="flex h-full flex-col">
      <header className="border-b border-paper-200 px-6 py-4 dark:border-night-800">
        <h2 className="font-display text-lg font-semibold text-ink-900 dark:text-paper-100">Ask your knowledge</h2>
        <p className="text-xs text-ink-500">Claude reads and writes your wiki through this chat</p>
        {(status === 'reconnecting' || status === 'disconnected') && (
          <p className="mt-1 text-xs text-accent-600 dark:text-accent-400">
            {status === 'reconnecting' ? 'Connection lost — reconnecting…' : 'Connection lost.'}
          </p>
        )}
      </header>
      <div className="flex-1 space-y-4 overflow-y-auto px-6 py-5">
        {messages.length === 0 && (
          <div className="flex h-full flex-col items-center justify-center text-center">
            <div className="font-display text-3xl text-ink-900 dark:text-paper-100">Thoth</div>
            <p className="mt-2 max-w-sm text-sm text-ink-500">
              Ask anything — “what did we decide in Tuesday's standup?” or
              “save this: the deploy uses port 9090”.
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
