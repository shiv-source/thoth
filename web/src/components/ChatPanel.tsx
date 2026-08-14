import { useEffect, useRef } from 'react'
import { useChat } from '../hooks/useChat'
import { ChatSocket } from '../ws/chat'
import { Composer } from './Composer'
import { MessageItem } from './MessageItem'

function createSocket(): ChatSocket {
  const socket = new ChatSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`)
  socket.connect()
  return socket
}

export function ChatPanel() {
  const socketRef = useRef<ChatSocket | null>(null)
  const socket = socketRef.current ?? (socketRef.current = createSocket())
  const { messages, streaming, send, cancel } = useChat(socket)
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streaming])

  return (
    <div className="flex h-full flex-col">
      <header className="border-b border-paper-200 px-6 py-4 dark:border-night-800">
        <h2 className="font-display text-lg font-semibold text-ink-900 dark:text-paper-100">Ask your knowledge</h2>
        <p className="text-xs text-ink-500">Claude reads and writes your wiki through this chat</p>
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
