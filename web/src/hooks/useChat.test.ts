import { act, renderHook } from '@testing-library/react'
import { afterAll, afterEach, describe, expect, it } from 'vitest'
import { ChatSocket } from '../ws/chat'
import { useChat } from './useChat'
import { FakeWS } from '../test/fakeWS'

const original = globalThis.WebSocket
globalThis.WebSocket = FakeWS as unknown as typeof WebSocket

function freshSocket(): ChatSocket {
  const socket = new ChatSocket('ws://x/ws')
  socket.connect()
  return socket
}

describe('useChat', () => {
  afterEach(() => { FakeWS.instances = [] })

  it('accumulates deltas into an assistant message', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    act(() => result.current.send('question'))
    expect(result.current.messages).toEqual([{ role: 'user', content: 'question' }])
    expect(result.current.streaming).toBe(true)

    const ws = FakeWS.instances[0]!
    const emit = (type: string, text: string) =>
      act(() => ws?.onmessage?.({ data: JSON.stringify({ type, text }) }))

    emit('assistant_delta', 'an')
    emit('assistant_delta', 'swer')
    emit('turn_done', '')

    expect(result.current.messages.at(-1)).toEqual({ role: 'assistant', content: 'answer' })
    expect(result.current.streaming).toBe(false)
  })

  it('records the conversation id from turn_done', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    const ws = FakeWS.instances[0]!
    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'turn_done', conversation_id: 'conv-9' }) }))

    expect(result.current.conversationId).toBe('conv-9')
  })

  it('renders error frames as a visible assistant message', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    const ws = FakeWS.instances[0]!
    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'error', message: 'cancelled' }) }))

    expect(result.current.messages.at(-1)).toEqual({ role: 'assistant', content: '⚠️ cancelled' })
    expect(result.current.streaming).toBe(false)
  })
})

afterAll(() => { globalThis.WebSocket = original })
