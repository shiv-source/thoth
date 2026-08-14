export type ServerMessage =
  | { type: 'assistant_start' }
  | { type: 'assistant_delta'; text: string }
  | { type: 'tool_activity'; tool: string; detail: string }
  | { type: 'turn_done' }
  | { type: 'error'; message: string }

export class ChatSocket {
  private ws: WebSocket | null = null
  private handler: (m: ServerMessage) => void = () => {}

  constructor(private readonly url: string) {}

  connect(): void {
    this.ws = new WebSocket(this.url)
    this.ws.onmessage = (e) => {
      // defensive: a malformed frame must not kill the UI
      try { this.handler(JSON.parse(e.data as string) as ServerMessage) } catch { /* ignore */ }
    }
  }

  onMessage(handler: (m: ServerMessage) => void): void { this.handler = handler }

  send(text: string): void { this.ws?.send(JSON.stringify({ type: 'send', text })) }
  cancel(): void { this.ws?.send(JSON.stringify({ type: 'cancel' })) }
  resume(conversationId: string): void { this.ws?.send(JSON.stringify({ type: 'resume', conversation_id: conversationId })) }
  close(): void { this.ws?.close() }
}
