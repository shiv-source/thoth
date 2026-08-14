export type ServerMessage =
  | { type: 'assistant_start' }
  | { type: 'assistant_delta'; text: string }
  | { type: 'tool_activity'; tool: string; detail: string }
  | { type: 'turn_done'; conversation_id?: string }
  | { type: 'error'; message: string }

export type ConnectionStatus = 'connected' | 'reconnecting' | 'disconnected'

export class ChatSocket {
  private ws: WebSocket | null = null
  private handler: (m: ServerMessage) => void = () => {}
  private statusHandler: (s: ConnectionStatus) => void = () => {}
  private conversationId: string | null = null
  private retried = false
  private closed = false
  private resumePending = false
  private retryTimer: ReturnType<typeof setTimeout> | null = null

  constructor(private readonly url: string) {}

  connect(): void {
    const ws = new WebSocket(this.url)
    this.ws = ws
    ws.onmessage = (e) => {
      // defensive: a malformed frame must not kill the UI
      try {
        const m = JSON.parse(e.data as string) as ServerMessage
        if (m.type === 'turn_done' && m.conversation_id) this.conversationId = m.conversation_id
        this.handler(m)
      } catch { /* ignore */ }
    }
    ws.onopen = () => {
      this.statusHandler('connected')
      // A freshly reconnected socket is CONNECTING: sending a resume frame
      // before it opens throws InvalidStateError, so resume from onopen.
      if (this.resumePending) {
        this.resumePending = false
        if (this.conversationId) this.resume(this.conversationId)
      }
    }
    ws.onclose = () => {
      if (this.closed) return
      // Auto-reconnect exactly once, then stay disconnected.
      if (this.retried) {
        this.resumePending = false
        this.statusHandler('disconnected')
        return
      }
      this.retried = true
      this.statusHandler('reconnecting')
      this.retryTimer = setTimeout(() => {
        this.resumePending = Boolean(this.conversationId)
        this.connect()
      }, 1000)
    }
  }

  onMessage(handler: (m: ServerMessage) => void): void { this.handler = handler }
  onStatusChange(handler: (s: ConnectionStatus) => void): void { this.statusHandler = handler }

  send(text: string): void { this.ws?.send(JSON.stringify({ type: 'send', text })) }
  cancel(): void { this.ws?.send(JSON.stringify({ type: 'cancel' })) }
  resume(conversationId: string): void {
    this.conversationId = conversationId
    this.ws?.send(JSON.stringify({ type: 'resume', conversation_id: conversationId }))
  }
  close(): void {
    this.closed = true
    if (this.retryTimer !== null) clearTimeout(this.retryTimer)
    this.ws?.close()
  }
}
