import { afterEach, describe, expect, it, vi } from 'vitest'
import { ChatSocket } from './chat'
import type { ServerMessage } from './protocol'
import { ServerEvent } from './events'
import { FakeWS } from '../test/fakeWS'

vi.stubGlobal('WebSocket', FakeWS)

describe('ChatSocket', () => {
    afterEach(() => {
        FakeWS.instances = []
    })

    it('sends typed frames and forwards parsed messages', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        const received: unknown[] = []
        socket.onMessage((m) => received.push(m))
        const ws = FakeWS.instances[0]!
        ws.open() // the handshake must complete before frames can be sent
        socket.send('hello')
        socket.cancel()

        expect(ws.sent[0]).toBe(JSON.stringify({ type: 'send', text: 'hello' }))
        expect(ws.sent[1]).toBe(JSON.stringify({ type: 'cancel' }))

        ws.onmessage!({ data: JSON.stringify({ type: 'assistant_delta', text: 'hi' }) })
        expect(received).toEqual([{ type: 'assistant_delta', text: 'hi' }])
    })

    it('forwards the wiki_changed push frame', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        const received: ServerMessage[] = []
        socket.onMessage((m) => received.push(m))
        const ws = FakeWS.instances[0]!
        ws.open()

        const frame: ServerMessage = {
            type: ServerEvent.WikiChanged,
            changes: [
                { op: 'write', path: 'notes/a.md' },
                { op: 'remove', path: 'old.md' }
            ]
        }
        ws.onmessage!({ data: JSON.stringify(frame) })
        expect(received).toEqual([frame])

        // The watcher's startup event carries no changes (omitempty on the
        // wire) and must parse as a bare wiki_changed frame.
        const bare: ServerMessage = { type: ServerEvent.WikiChanged }
        ws.onmessage!({ data: JSON.stringify(bare) })
        expect(received).toEqual([frame, bare])
    })

    it('drops frames that parse as JSON but fail the schema', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        const received: unknown[] = []
        socket.onMessage((m) => received.push(m))
        const ws = FakeWS.instances[0]!
        ws.open()

        // JSON-valid but not a ServerMessage: an unknown type, and a known
        // type missing its required field — neither may reach the handler.
        ws.onmessage!({ data: JSON.stringify({ type: 'bogus_event' }) })
        ws.onmessage!({ data: JSON.stringify({ type: 'assistant_delta' }) })
        expect(received).toEqual([])
    })

    it('reconnects once after a drop and resumes the conversation', () => {
        vi.useFakeTimers()
        try {
            const socket = new ChatSocket('ws://x/ws')
            const statuses: string[] = []
            socket.onStatusChange((s) => statuses.push(s))
            socket.connect()
            socket.onMessage(() => {})

            // The first turn finishes and tells us the conversation id.
            FakeWS.instances[0]!.onmessage!({ data: JSON.stringify({ type: 'turn_done', conversation_id: 'conv-1' }) })

            FakeWS.instances[0]!.onclose!()
            expect(statuses).toContain('reconnecting')

            vi.advanceTimersByTime(1000)
            expect(FakeWS.instances).toHaveLength(2)
            // The fresh socket is still CONNECTING: no frame may be sent yet
            // (real WebSockets throw InvalidStateError on send()).
            expect(FakeWS.instances[1]!.sent).toEqual([])

            // Once the handshake completes, the reconnect resumes the conversation.
            FakeWS.instances[1]!.open()
            expect(FakeWS.instances[1]!.sent).toEqual([JSON.stringify({ type: 'resume', conversation_id: 'conv-1' })])

            // A second drop stays disconnected: exactly one retry.
            FakeWS.instances[1]!.onclose!()
            expect(FakeWS.instances).toHaveLength(2)
            expect(statuses).toContain('disconnected')
        } finally {
            vi.useRealTimers()
        }
    })

    it('does not reconnect after close()', () => {
        vi.useFakeTimers()
        try {
            const socket = new ChatSocket('ws://x/ws')
            socket.connect()
            socket.close()
            FakeWS.instances[0]!.onclose!()
            vi.advanceTimersByTime(1000)
            expect(FakeWS.instances).toHaveLength(1)
        } finally {
            vi.useRealTimers()
        }
    })

    it('reports connected on open', () => {
        const socket = new ChatSocket('ws://x/ws')
        const statuses: string[] = []
        socket.onStatusChange((s) => statuses.push(s))
        socket.connect()
        FakeWS.instances[0]!.onopen!()
        expect(statuses).toEqual(['connected'])
    })

    it('sends the open frame for a loaded conversation', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        const ws = FakeWS.instances[0]!
        ws.open()
        socket.open('conv-9')
        expect(ws.sent).toEqual([JSON.stringify({ type: 'open', conversation_id: 'conv-9' })])
    })

    it('defers the open frame until the handshake completes', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        // Deep links call open() right after connect(): the socket is still
        // CONNECTING and a real WebSocket would throw on send.
        socket.open('conv-9')
        const ws = FakeWS.instances[0]!
        expect(ws.sent).toEqual([])

        ws.open()
        expect(ws.sent).toEqual([JSON.stringify({ type: 'open', conversation_id: 'conv-9' })])
    })

    it('drops a deferred open frame when the socket is closed', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        socket.open('conv-9')
        socket.close()
        const ws = FakeWS.instances[0]!
        ws.onopen!()
        expect(ws.sent).toEqual([])
    })
})

describe('ChatSocket newChat', () => {
    afterEach(() => {
        FakeWS.instances = []
    })

    it('sends the new_chat frame when connected', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        const ws = FakeWS.instances[0]!
        ws.open()
        socket.newChat()
        expect(ws.sent).toEqual([JSON.stringify({ type: 'new_chat' })])
    })

    it('defers the new_chat frame until the handshake completes', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        socket.newChat()
        const ws = FakeWS.instances[0]!
        expect(ws.sent).toEqual([])
        ws.open()
        expect(ws.sent).toEqual([JSON.stringify({ type: 'new_chat' })])
    })

    it('clears the resume id so a reconnect cannot resurrect the old pin', () => {
        vi.useFakeTimers()
        try {
            const socket = new ChatSocket('ws://x/ws')
            socket.connect()
            FakeWS.instances[0]!.open()
            FakeWS.instances[0]!.onmessage!({ data: JSON.stringify({ type: 'turn_done', conversation_id: 'conv-1' }) })

            socket.newChat()
            FakeWS.instances[0]!.onclose!()
            vi.advanceTimersByTime(1000)
            FakeWS.instances[1]!.open()
            // No resume (or any other) frame may leave the new socket: the pin
            // was dropped by new_chat.
            expect(FakeWS.instances[1]!.sent).toEqual([])
        } finally {
            vi.useRealTimers()
        }
    })

    it('discards a deferred open when newChat wins', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        socket.open('conv-9') // CONNECTING: deferred
        socket.newChat() // must supersede the deferred open
        const ws = FakeWS.instances[0]!
        ws.open()
        expect(ws.sent).toEqual([JSON.stringify({ type: 'new_chat' })])
    })
})

describe('ChatSocket presence', () => {
    afterEach(() => {
        FakeWS.instances = []
    })

    it('sends the presence frame when connected', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        const ws = FakeWS.instances[0]!
        ws.open()
        socket.setPresence(false)
        expect(ws.sent).toEqual([JSON.stringify({ type: 'presence', active: false })])
        socket.setPresence(true)
        expect(ws.sent).toEqual([
            JSON.stringify({ type: 'presence', active: false }),
            JSON.stringify({ type: 'presence', active: true })
        ])
    })

    it('defers the presence frame until the handshake completes', () => {
        const socket = new ChatSocket('ws://x/ws')
        socket.connect()
        socket.setPresence(false)
        const ws = FakeWS.instances[0]!
        expect(ws.sent).toEqual([])
        ws.open()
        expect(ws.sent).toEqual([JSON.stringify({ type: 'presence', active: false })])
    })

    it('re-sends the last presence after a reconnect', () => {
        vi.useFakeTimers()
        try {
            const socket = new ChatSocket('ws://x/ws')
            socket.connect()
            FakeWS.instances[0]!.open()
            socket.setPresence(false)
            FakeWS.instances[0]!.onclose!()
            vi.advanceTimersByTime(1000)
            FakeWS.instances[1]!.open()
            // A hidden tab stays counted as away across the reconnect.
            expect(FakeWS.instances[1]!.sent).toEqual([JSON.stringify({ type: 'presence', active: false })])
        } finally {
            vi.useRealTimers()
        }
    })
})
