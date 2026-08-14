import { useCallback, useEffect, useRef, useState } from 'react'
import { ChatSocket, type ServerMessage } from '../ws/chat'

export interface ChatMessage { role: 'user' | 'assistant'; content: string }

export function useChat(socket: ChatSocket | null) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [streaming, setStreaming] = useState(false)
  const [conversationId, setConversationId] = useState<string | null>(null)
  const [lastTool, setLastTool] = useState<string | null>(null)
  const messagesRef = useRef<ChatMessage[]>([])
  const streamingRef = useRef(false)
  const conversationIdRef = useRef<string | null>(null)

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
    socket?.send(text)
  }, [push, socket])

  const cancel = useCallback(() => {
    socket?.cancel()
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
      case 'tool_activity':
        // The detail carries the raw tool-input JSON; surface the path it
        // reads/writes (e.g. "meetings/…") and fall back to the tool name.
        setLastTool(toolLabel(m.tool, m.detail))
        break
      case 'turn_done':
        // The server sends the conversation id on every finished turn; keep
        // it so a reconnect can resume this conversation.
        if (m.conversation_id) {
          conversationIdRef.current = m.conversation_id
          setConversationId(m.conversation_id)
        }
        streamingRef.current = false
        setStreaming(false)
        setLastTool(null)
        break
      case 'error':
        // Surface cancelled/crash feedback as a visible assistant message so
        // the user knows the turn did not complete.
        push({ role: 'assistant', content: `⚠️ ${m.message}` })
        streamingRef.current = false
        setStreaming(false)
        setLastTool(null)
        break
      default:
        break
    }
  }, [appendAssistant, push])

  // load replaces the whole conversation with history fetched from the server.
  // Local only — the caller pins the server side via socket.open(conversationId).
  const load = useCallback((msgs: ChatMessage[], convId: string) => {
    messagesRef.current = msgs
    setMessages(msgs)
    streamingRef.current = false
    setStreaming(false)
    conversationIdRef.current = convId
    setConversationId(convId)
    setLastTool(null)
  }, [])

  const reset = useCallback(() => {
    // Local only — no frame goes to the server; it creates a fresh
    // conversation on the next send anyway.
    messagesRef.current = []
    setMessages([])
    streamingRef.current = false
    setStreaming(false)
    conversationIdRef.current = null
    setConversationId(null)
    setLastTool(null)
  }, [])

  useEffect(() => {
    if (socket) socket.onMessage(handle)
  }, [socket, handle])

  return { messages, streaming, conversationId, lastTool, send, cancel, load, reset }
}

/** Pick the label for the tool status line: a path from the input JSON when
 *  present, otherwise the tool name itself. */
function toolLabel(tool: string, detail: string): string {
  try {
    const input = JSON.parse(detail) as { path?: unknown }
    if (typeof input.path === 'string' && input.path) return input.path
  } catch {
    // detail is not JSON — fall through to the tool name
  }
  return tool
}
