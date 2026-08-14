import { useCallback, useRef, useState } from 'react'
import { ChatSocket, type ServerMessage } from '../ws/chat'

export interface ChatMessage { role: 'user' | 'assistant'; content: string }

export function useChat(socket: ChatSocket) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [streaming, setStreaming] = useState(false)
  const messagesRef = useRef<ChatMessage[]>([])
  const streamingRef = useRef(false)

  const push = useCallback((m: ChatMessage) => {
    messagesRef.current = [...messagesRef.current, m]
    setMessages(messagesRef.current)
  }, [])

  const appendAssistant = useCallback((delta: string) => {
    const cur = messagesRef.current
    const last = cur[cur.length - 1]
    if (last && last.role === 'assistant') {
      const next = [...cur.slice(0, -1), { ...last, content: last.content + delta }]
      messagesRef.current = next
    } else {
      messagesRef.current = [...cur, { role: 'assistant', content: delta }]
    }
    setMessages(messagesRef.current)
  }, [])

  const send = useCallback((text: string) => {
    push({ role: 'user', content: text })
    streamingRef.current = true
    setStreaming(true)
    socket.send(text)
  }, [push, socket])

  const cancel = useCallback(() => {
    socket.cancel()
    streamingRef.current = false
    setStreaming(false)
  }, [socket])

  const handle = useCallback((m: ServerMessage) => {
    switch (m.type) {
      case 'assistant_start':
        streamingRef.current = true
        setStreaming(true)
        break
      case 'assistant_delta':
        appendAssistant(m.text)
        break
      case 'turn_done':
      case 'error':
        streamingRef.current = false
        setStreaming(false)
        break
      default:
        break
    }
  }, [appendAssistant])

  socket.onMessage(handle)

  return { messages, streaming, send, cancel }
}
