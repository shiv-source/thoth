import { afterEach, describe, expect, it, vi } from 'vitest'
import { ChatSocket } from './chat'
import { FakeWS } from '../test/fakeWS'

vi.stubGlobal('WebSocket', FakeWS)

describe('ChatSocket', () => {
  afterEach(() => { FakeWS.instances = [] })

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
})
