import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { Provider } from 'react-redux'
import { afterAll, afterEach, describe, expect, it, vi } from 'vitest'
// Import the axios double before anything that transitively pulls axios
// (./useChat → ../store → api/client): the vi.mock factory below runs the
// moment axios is first required.
import { axiosModuleMock, stubAPI } from '../test/mockAxios'
import { ChatSocket } from '../ws/chat'
import { useChat } from './useChat'
import { FakeWS } from '../test/fakeWS'
import { makeStore } from '../store'

// useChat dispatches fetchTree on wiki_changed frames, which calls the REST
// client — the axios double keeps that off the real transport.
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const original = globalThis.WebSocket
globalThis.WebSocket = FakeWS as unknown as typeof WebSocket

function freshSocket(): ChatSocket {
    const socket = new ChatSocket('ws://x/ws/v1')
    socket.connect()
    FakeWS.instances[0]!.open() // handshake completes so sends do not throw
    return socket
}

// useChat reads from the chat slice, so it must render inside a Provider;
// a fresh store per hook keeps the tests isolated.
function renderChatHook(socket: ChatSocket) {
    const store = makeStore()
    const wrapper = ({ children }: { children: ReactNode }) => <Provider store={store}>{children}</Provider>
    return renderHook(() => useChat(socket), { wrapper })
}

describe('useChat', () => {
    afterEach(() => {
        FakeWS.instances = []
    })

    it('accumulates deltas into an assistant message', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

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
        const { result } = renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'turn_done', conversation_id: 'conv-9' }) }))

        expect(result.current.conversationId).toBe('conv-9')
    })

    it('surfaces token usage from turn_done', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        act(() =>
            ws?.onmessage?.({
                data: JSON.stringify({
                    type: 'turn_done',
                    conversation_id: 'conv-9',
                    usage: { input_tokens: 8, output_tokens: 2, cache_read_tokens: 0, cache_write_tokens: 0 }
                })
            })
        )

        expect(result.current.lastUsage).toEqual({
            input_tokens: 8,
            output_tokens: 2,
            cache_read_tokens: 0,
            cache_write_tokens: 0
        })
    })

    it('renders error frames as a visible assistant message', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'error', message: 'cancelled' }) }))

        expect(result.current.messages.at(-1)).toEqual({ role: 'assistant', content: '⚠️ cancelled' })
        expect(result.current.streaming).toBe(false)
    })

    it('tracks the tool in use and clears it when the turn ends', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        const emit = (frame: string) => act(() => ws?.onmessage?.({ data: frame }))

        // The detail is raw tool-input JSON; the path it reads becomes the label.
        emit(
            JSON.stringify({
                type: 'tool_activity',
                tool: 'Read',
                detail: JSON.stringify({ path: 'meetings/standup.md' })
            })
        )
        expect(result.current.lastTool).toBe('meetings/standup.md')

        emit(JSON.stringify({ type: 'turn_done' }))
        expect(result.current.lastTool).toBeNull()
    })

    it('falls back to the tool name when the detail is not a path JSON', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        act(() =>
            ws?.onmessage?.({ data: JSON.stringify({ type: 'tool_activity', tool: 'Bash', detail: 'not json' }) })
        )

        expect(result.current.lastTool).toBe('Bash')
    })

    it('clears the tool on error frames', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'tool_activity', tool: 'Read', detail: '{}' }) }))
        expect(result.current.lastTool).toBe('Read')

        act(() => ws?.onmessage?.({ data: JSON.stringify({ type: 'error', message: 'cancelled' }) }))
        expect(result.current.lastTool).toBeNull()
    })

    it('load() replaces messages and the conversation id, resetting streaming state', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        act(() => result.current.send('hello'))
        act(() =>
            result.current.load(
                [
                    { role: 'user', content: 'old question' },
                    { role: 'assistant', content: 'old answer' }
                ],
                'conv-7'
            )
        )

        expect(result.current.messages).toEqual([
            { role: 'user', content: 'old question' },
            { role: 'assistant', content: 'old answer' }
        ])
        expect(result.current.conversationId).toBe('conv-7')
        expect(result.current.streaming).toBe(false)
        expect(result.current.lastTool).toBeNull()
        expect(result.current.lastUsage).toBeNull()
        // load is local-only: no frame left the socket
        expect(FakeWS.instances[0]!.sent).toEqual([JSON.stringify({ type: 'send', text: 'hello' })])
    })

    it('load() restores lastUsage from persisted history', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

        act(() =>
            result.current.load([{ role: 'assistant', content: 'persisted answer' }], 'conv-7', {
                input_tokens: 8,
                output_tokens: 2,
                cache_read_tokens: 5,
                cache_write_tokens: 0
            })
        )

        expect(result.current.lastUsage).toEqual({
            input_tokens: 8,
            output_tokens: 2,
            cache_read_tokens: 5,
            cache_write_tokens: 0
        })
    })

    it('refetches the wiki tree when a wiki_changed frame arrives', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => ({ nodes: [] }) })
        const socket = freshSocket()
        renderChatHook(socket)

        const ws = FakeWS.instances[0]!
        act(() =>
            ws?.onmessage?.({
                data: JSON.stringify({ type: 'wiki_changed', changes: [{ op: 'write', path: 'notes/a.md' }] })
            })
        )

        await waitFor(() => expect(mocks.get).toHaveBeenCalledWith('/api/v1/wiki/tree'))
        expect(mocks.get).toHaveBeenCalledTimes(1)
    })

    it('reset() clears locally and unpins the server', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)

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
        expect(ws.sent).toEqual([JSON.stringify({ type: 'send', text: 'hello' }), JSON.stringify({ type: 'new_chat' })])
    })
})

afterAll(() => {
    globalThis.WebSocket = original
})

describe('useChat thinking state', () => {
    it('shows thinking from assistant_start until the first delta', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)
        const ws = FakeWS.instances[0]!
        const emit = (type: string, extra: Record<string, unknown> = {}) =>
            act(() => ws?.onmessage?.({ data: JSON.stringify({ type, ...extra }) }))

        emit('assistant_start')
        expect(result.current.thinking).toBe(true)

        act(() =>
            ws?.onmessage?.({ data: JSON.stringify({ type: 'assistant_thinking', text: 'checking the folder' }) })
        )
        expect(result.current.thinking).toBe(true)
        expect(result.current.thinkingText).toBe('checking the folder')

        emit('assistant_delta', { text: '' })
        expect(result.current.thinking).toBe(false)
    })

    it('clears thinking on tool activity, turn_done, and errors', () => {
        const socket = freshSocket()
        const { result } = renderChatHook(socket)
        const ws = FakeWS.instances[0]!
        const emit = (type: string, extra: Record<string, unknown> = {}) =>
            act(() => ws?.onmessage?.({ data: JSON.stringify({ type, ...extra }) }))

        emit('assistant_start')
        emit('tool_activity', { tool: 'Read', detail: '{}' })
        expect(result.current.thinking).toBe(false)

        emit('assistant_start')
        emit('turn_done')
        expect(result.current.thinking).toBe(false)

        emit('assistant_start')
        emit('error', { message: 'cancelled' })
        expect(result.current.thinking).toBe(false)
    })
})
