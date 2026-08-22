import { ClientEvent, ServerEvent } from './events'
import { serverMessageSchema, type ServerMessage } from './protocol'

export type ConnectionStatus = 'connected' | 'reconnecting' | 'disconnected'

export class ChatSocket {
    private ws: WebSocket | null = null
    private handler: (m: ServerMessage) => void = () => {}
    private statusHandler: (s: ConnectionStatus) => void = () => {}
    private conversationId: string | null = null
    private retried = false
    private closed = false
    private resumePending = false
    private openPending: string | null = null
    private newChatPending = false
    private presence: boolean | null = null
    private retryTimer: ReturnType<typeof setTimeout> | null = null

    constructor(private readonly url: string) {}

    connect(): void {
        const ws = new WebSocket(this.url)
        this.ws = ws
        ws.onmessage = (e) => {
            // defensive: a malformed frame must not kill the UI
            try {
                const parsed = serverMessageSchema.safeParse(JSON.parse(e.data as string))
                if (!parsed.success) {
                    // Not a valid ServerMessage — drop it. The schema checks
                    // both the frame type and every field it carries.
                    return
                }
                const m = parsed.data
                if (m.type === ServerEvent.TurnDone && m.conversation_id) this.conversationId = m.conversation_id
                this.handler(m)
            } catch {
                /* ignore */
            }
        }
        ws.onopen = () => {
            this.statusHandler('connected')
            // A freshly reconnected socket is CONNECTING: sending a resume frame
            // before it opens throws InvalidStateError, so resume from onopen.
            if (this.resumePending) {
                this.resumePending = false
                if (this.conversationId) this.resume(this.conversationId)
            }
            if (this.openPending) {
                const id = this.openPending
                this.openPending = null
                this.ws?.send(JSON.stringify({ type: ClientEvent.Open, conversation_id: id }))
            }
            if (this.newChatPending) {
                this.newChatPending = false
                this.ws?.send(JSON.stringify({ type: ClientEvent.NewChat }))
            }
            // A reconnect is treated as active by the server; re-send the last
            // presence so a hidden tab stays counted as away across reconnects.
            if (this.presence !== null) {
                this.ws?.send(JSON.stringify({ type: ClientEvent.Presence, active: this.presence }))
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

    onMessage(handler: (m: ServerMessage) => void): void {
        this.handler = handler
    }
    onStatusChange(handler: (s: ConnectionStatus) => void): void {
        this.statusHandler = handler
    }

    send(text: string): void {
        this.ws?.send(JSON.stringify({ type: ClientEvent.Send, text }))
    }
    cancel(): void {
        this.ws?.send(JSON.stringify({ type: ClientEvent.Cancel }))
    }
    resume(conversationId: string): void {
        this.conversationId = conversationId
        this.ws?.send(JSON.stringify({ type: ClientEvent.Resume, conversation_id: conversationId }))
    }
    // open pins the server-side conversation for the next send — unlike resume,
    // it does NOT replay anything, so it must not become the reconnect-resume
    // id (a reconnect would then replay the loaded history over the UI).
    // Frames sent before the handshake completes throw InvalidStateError, so
    // defer until onopen (deep links call open() right after connect()).
    open(conversationId: string): void {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: ClientEvent.Open, conversation_id: conversationId }))
            return
        }
        this.openPending = conversationId
    }

    // newChat unpins the server-side conversation and drops every queued
    // resume/open so nothing can resurrect the old pin (a reconnect-resume
    // would otherwise re-pin the old conversation). Deferred like open() when
    // the handshake has not completed.
    newChat(): void {
        this.conversationId = null
        this.resumePending = false
        this.openPending = null
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: ClientEvent.NewChat }))
            return
        }
        this.newChatPending = true
    }

    // setPresence reports whether the tab is visible/foreground. A hidden tab
    // is not an active chat client, so the server flushes idle pooled CLI
    // processes after its relaxation timeout. Deferred like open() when the
    // handshake has not completed; re-sent on reconnect.
    setPresence(active: boolean): void {
        this.presence = active
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: ClientEvent.Presence, active }))
        }
    }
    close(): void {
        this.closed = true
        this.openPending = null
        this.newChatPending = false
        if (this.retryTimer !== null) clearTimeout(this.retryTimer)
        this.ws?.close()
    }
}
