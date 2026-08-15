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
  FakeWS.instances[0]!.open() // handshake completes so sends do not throw
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

  it('tracks the tool in use and clears it when the turn ends', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    const ws = FakeWS.instances[0]!
    const emit = (frame: string) => act(() => ws?.onmessage?.({ data: frame }))

    // The detail is raw tool-input JSON; the path it reads becomes the label.
    emit(JSON.stringify({ type: 'tool_activity', tool: 'Read', detail: JSON.stringify({ path: 'meetings/standup.md' }) }))
    expect(result.current.lastTool).toBe('meetings/standup.md')

    emit(JSON.stringify({ type: 'turn_done' }))
    expect(result.current.lastTool).toBeNull()
  })

  it('falls back to the tool name when the detail is not a path JSON', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    const ws = FakeWS.instances[0]!
    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'tool_activity', tool: 'Bash', detail: 'not json' }) }))

    expect(result.current.lastTool).toBe('Bash')
  })

  it('clears the tool on error frames', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    const ws = FakeWS.instances[0]!
    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'tool_activity', tool: 'Read', detail: '{}' }) }))
    expect(result.current.lastTool).toBe('Read')

    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'error', message: 'cancelled' }) }))
    expect(result.current.lastTool).toBeNull()
  })

  it('load() replaces messages and the conversation id, resetting streaming state', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    act(() => result.current.send('hello'))
    act(() => result.current.load([
      { role: 'user', content: 'old question' },
      { role: 'assistant', content: 'old answer' },
    ], 'conv-7'))

    expect(result.current.messages).toEqual([
      { role: 'user', content: 'old question' },
      { role: 'assistant', content: 'old answer' },
    ])
    expect(result.current.conversationId).toBe('conv-7')
    expect(result.current.streaming).toBe(false)
    expect(result.current.lastTool).toBeNull()
    // load is local-only: no frame left the socket
    expect(FakeWS.instances[0]!.sent).toEqual([JSON.stringify({ type: 'send', text: 'hello' })])
  })

  it('reset() clears locally and unpins the server', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))

    const ws = FakeWS.instances[0]!
    act(() => result.current.send('hello'))
    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'turn_done', conversation_id: 'conv-9' }) }))
    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'tool_activity', tool: 'Read', detail: '{}' }) }))

    act(() => result.current.reset())

    expect(result.current.messages).toEqual([])
    expect(result.current.streaming).toBe(false)
    expect(result.current.conversationId).toBeNull()
    expect(result.current.lastTool).toBeNull()
    // The unpin frame must reach the server, or the next send would
    // continue the old pinned conversation.
    expect(ws.sent).toEqual([
      JSON.stringify({ type: 'send', text: 'hello' }),
      JSON.stringify({ type: 'new_chat' }),
    ])
  })
})

afterAll(() => { globalThis.WebSocket = original })

describe('useChat thinking state', () => {
  it('shows thinking from assistant_start until the first delta', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))
    const ws = FakeWS.instances[0]!
    const emit = (type: string) =>
      act(() => ws?.onmessage?.({ data: JSON.stringify({ type }) }))

    emit('assistant_start')
    expect(result.current.thinking).toBe(true)

    act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'assistant_thinking', text: 'checking the folder' }) }))
    expect(result.current.thinking).toBe(true)
    expect(result.current.thinkingText).toBe('checking the folder')

    emit('assistant_delta')
    expect(result.current.thinking).toBe(false)
  })

  it('clears thinking on tool activity, turn_done, and errors', () => {
    const socket = freshSocket()
    const { result } = renderHook(() => useChat(socket))
    const ws = FakeWS.instances[0]!
    const emit = (type: string) =>
      act(() => ws?.onmessage?.({ data: JSON.stringify({ type }) }))

    emit('assistant_start')
    emit('tool_activity')
    expect(result.current.thinking).toBe(false)

    emit('assistant_start')
    emit('turn_done')
    expect(result.current.thinking).toBe(false)

    emit('assistant_start')
    emit('error')
    expect(result.current.thinking).toBe(false)
  })
})
