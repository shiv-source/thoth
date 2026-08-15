import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Conversation } from '../../api/client'
import { axiosError, axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import {
    deleteConversation,
    fetchConversations,
    selectConversations,
    selectConversationsLoading
} from './conversationsSlice'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const convs: Conversation[] = [
    { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1', title: 'Today chat', created_at: '2026-08-15T09:00:00Z' },
    { id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb2', title: 'Older chat', created_at: '2026-08-14T09:00:00Z' }
]

describe('conversationsSlice', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('starts with an empty, loading state', () => {
        const store = makeStore()
        expect(store.getState().conversations).toEqual({ list: null, loading: true, error: null })
    })

    it('loads the conversation list', async () => {
        stubAPI(mocks, { 'GET /api/conversations': () => ({ conversations: convs }) })
        const store = makeStore()
        await store.dispatch(fetchConversations())
        expect(selectConversations(store.getState()).list).toEqual(convs)
        expect(selectConversationsLoading(store.getState())).toBe(false)
        expect(selectConversations(store.getState()).error).toBeNull()
    })

    it('tracks the in-flight fetch', async () => {
        let resolveGet!: (r: { data: { conversations: Conversation[] } }) => void
        mocks.get.mockReturnValueOnce(
            new Promise<{ data: { conversations: Conversation[] } }>((resolve) => {
                resolveGet = resolve
            })
        )
        const store = makeStore()
        const pending = store.dispatch(fetchConversations())
        expect(selectConversationsLoading(store.getState())).toBe(true)
        resolveGet({ data: { conversations: convs } })
        await pending
        expect(selectConversationsLoading(store.getState())).toBe(false)
    })

    it('sets an error message when the fetch fails', async () => {
        mocks.get.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await store.dispatch(fetchConversations())
        const s = selectConversations(store.getState())
        expect(s.error).toBe('could not load conversations')
        expect(s.loading).toBe(false)
    })

    it('removes a deleted conversation from the list', async () => {
        stubAPI(mocks, {
            'GET /api/conversations': () => ({ conversations: convs }),
            'DELETE /api/conversations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1': () => undefined
        })
        const store = makeStore()
        await store.dispatch(fetchConversations())
        await store.dispatch(deleteConversation('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1'))
        expect(selectConversations(store.getState()).list).toEqual([convs[1]])
    })

    it('keeps the list when the delete fails', async () => {
        stubAPI(mocks, { 'GET /api/conversations': () => ({ conversations: convs }) })
        mocks.delete.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await store.dispatch(fetchConversations())
        await expect(
            store.dispatch(deleteConversation('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')).unwrap()
        ).rejects.toThrow()
        expect(selectConversations(store.getState()).list).toEqual(convs)
    })
})
