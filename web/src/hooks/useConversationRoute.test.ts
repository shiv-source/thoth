import { act, renderHook } from '@testing-library/react'
import { useCallback, useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api, type TokenUsage } from '../api/client'
import type { ChatSocket } from '../ws/chat'
import type { ChatMessage } from './useChat'
import { useConversationRoute } from './useConversationRoute'

vi.mock('../api/client', () => ({
    api: { getConversation: vi.fn() }
}))
const getConversation = vi.mocked(api.getConversation)

const ID_A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const ID_B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'

function message(content: string) {
    return { id: 1, conversation_id: 'x', role: 'user' as const, content, created_at: '2026-08-15T00:00:00Z' }
}

// Emulates ChatPage: load/reset update the same state the hook receives as
// conversationId (like useChat does), so URL→state round trips are faithful.
function renderRoute(initialId: string | null = null) {
    const open = vi.fn()
    const socket = { open } as unknown as ChatSocket
    const load = vi.fn()
    const reset = vi.fn()
    const onError = vi.fn()
    const utils = renderHook(() => {
        const [conv, setConv] = useState(initialId)
        const loadImpl = useCallback(
            (msgs: ChatMessage[], cid: string, usage?: TokenUsage) => {
                load(msgs, cid, usage)
                setConv(cid)
            },
            [load]
        )
        const resetImpl = useCallback(() => {
            reset()
            setConv(null)
        }, [reset])
        useConversationRoute({ socket, conversationId: conv, load: loadImpl, reset: resetImpl, onError })
        return { conv, setConv }
    })
    return { ...utils, open, load, reset, onError }
}

const flush = () => act(async () => {})

describe('useConversationRoute', () => {
    beforeEach(() => {
        window.history.pushState(null, '', '/')
        getConversation.mockReset()
    })

    it('loads the deep-linked conversation on mount', async () => {
        window.history.pushState(null, '', `/chat/${ID_A}`)
        getConversation.mockResolvedValue({ conversation: {} as never, messages: [message('hi')] })
        const { open, load } = renderRoute()

        await flush()
        expect(getConversation).toHaveBeenCalledWith(ID_A)
        expect(load).toHaveBeenCalledWith([{ role: 'user', content: 'hi' }], ID_A, undefined)
        expect(open).toHaveBeenCalledWith(ID_A)
        expect(window.location.pathname).toBe(`/chat/${ID_A}`)
    })

    it('passes the persisted last-message usage to load', async () => {
        window.history.pushState(null, '', `/chat/${ID_A}`)
        getConversation.mockResolvedValue({
            conversation: {} as never,
            messages: [
                {
                    id: 2,
                    conversation_id: ID_A,
                    role: 'assistant' as const,
                    content: 'answer',
                    created_at: '2026-08-15T00:00:00Z',
                    usage: { input_tokens: 8, output_tokens: 2, cache_read_tokens: 0, cache_write_tokens: 0 }
                }
            ]
        })
        const { load } = renderRoute()
        await flush()
        expect(load).toHaveBeenCalledWith([{ role: 'assistant', content: 'answer' }], ID_A, {
            input_tokens: 8,
            output_tokens: 2,
            cache_read_tokens: 0,
            cache_write_tokens: 0
        })
    })

    it('stays a fresh chat on the root path', async () => {
        const { load, reset } = renderRoute()

        await flush()
        expect(getConversation).not.toHaveBeenCalled()
        expect(load).not.toHaveBeenCalled()
        expect(reset).not.toHaveBeenCalled()
    })

    it('falls back to a fresh chat when the deep link id is unknown', async () => {
        window.history.pushState(null, '', `/chat/${ID_A}`)
        getConversation.mockRejectedValue(new Error('404 Not Found'))
        const { onError, reset } = renderRoute()

        await flush()
        expect(onError).toHaveBeenCalledWith('Conversation not found')
        expect(window.location.pathname).toBe('/chat')
        expect(reset).toHaveBeenCalled()
    })

    it('pushes /chat/<id> when the conversation id changes', async () => {
        const { result } = renderRoute()
        await flush()

        // turn_done in useChat sets the conversation id.
        act(() => result.current.setConv(ID_A))
        expect(window.location.pathname).toBe(`/chat/${ID_A}`)

        // New chat resets the URL to the fresh-chat path.
        act(() => result.current.setConv(null))
        expect(window.location.pathname).toBe('/chat')
    })

    it('loads the conversation the browser navigates back to', async () => {
        window.history.pushState(null, '', `/chat/${ID_A}`)
        const { load } = renderRoute(ID_A)
        await flush()

        getConversation.mockResolvedValue({ conversation: {} as never, messages: [message('back')] })
        act(() => {
            window.history.pushState(null, '', `/chat/${ID_B}`)
            window.dispatchEvent(new PopStateEvent('popstate'))
        })
        await flush()

        expect(getConversation).toHaveBeenCalledWith(ID_B)
        expect(load).toHaveBeenCalledWith([{ role: 'user', content: 'back' }], ID_B, undefined)
        expect(window.location.pathname).toBe(`/chat/${ID_B}`)
    })
})
