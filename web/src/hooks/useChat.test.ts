import { act, renderHook } from '@testing-library/react'
import { afterAll, describe, expect, it } from 'vitest'
import { ChatSocket } from '../ws/chat'
import { useChat } from './useChat'
import { FakeWS } from '../test/fakeWS'

const original = globalThis.WebSocket
globalThis.WebSocket = FakeWS as unknown as typeof WebSocket

describe('useChat', () => {
  it('accumulates deltas into an assistant message', () => {
    const socket = new ChatSocket('ws://x/ws')
    socket.connect()
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
})

afterAll(() => { globalThis.WebSocket = original })
