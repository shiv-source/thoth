import { describe, expect, it, vi } from 'vitest'
import { ChatSocket } from './chat'
import { FakeWS } from '../test/fakeWS'

vi.stubGlobal('WebSocket', FakeWS)

describe('ChatSocket', () => {
  it('sends typed frames and forwards parsed messages', () => {
    const socket = new ChatSocket('ws://x/ws')
    socket.connect()
    const received: unknown[] = []
    socket.onMessage((m) => received.push(m))
    socket.send('hello')
    socket.cancel()

    const ws = FakeWS.instances[0]!
    expect(ws.sent[0]).toBe(JSON.stringify({ type: 'send', text: 'hello' }))
    expect(ws.sent[1]).toBe(JSON.stringify({ type: 'cancel' }))

    ws.onmessage!({ data: JSON.stringify({ type: 'assistant_delta', text: 'hi' }) })
    expect(received).toEqual([{ type: 'assistant_delta', text: 'hi' }])
  })
})
