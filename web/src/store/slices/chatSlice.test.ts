import { beforeEach, describe, expect, it } from 'vitest'
import { makeStore } from '../index'
import {
    assistantDelta,
    assistantStart,
    assistantThinking,
    chatError,
    loadChat,
    resetChat,
    selectConversationId,
    selectLastTool,
    selectLastUsage,
    selectMessages,
    selectStreaming,
    selectThinking,
    selectThinkingText,
    stopStreaming,
    toolActivity,
    turnDone,
    userMessage
} from './chatSlice'

describe('chatSlice', () => {
    let store: ReturnType<typeof makeStore>
    beforeEach(() => {
        store = makeStore()
    })

    it('starts empty and idle', () => {
        expect(selectMessages(store.getState())).toEqual([])
        expect(selectStreaming(store.getState())).toBe(false)
        expect(selectConversationId(store.getState())).toBeNull()
        expect(selectLastTool(store.getState())).toBeNull()
        expect(selectThinking(store.getState())).toBe(false)
        expect(selectThinkingText(store.getState())).toBe('')
        expect(selectLastUsage(store.getState())).toBeNull()
    })

    it('pushes the user message and starts streaming on send', () => {
        store.dispatch(userMessage('hello wiki'))
        expect(selectMessages(store.getState())).toEqual([{ role: 'user', content: 'hello wiki' }])
        expect(selectStreaming(store.getState())).toBe(true)
    })

    it('starts thinking on assistant_start', () => {
        store.dispatch(assistantStart())
        expect(selectStreaming(store.getState())).toBe(true)
        expect(selectThinking(store.getState())).toBe(true)
    })

    it('shows the thinking text', () => {
        store.dispatch(assistantThinking('checking the inbox folder'))
        expect(selectThinking(store.getState())).toBe(true)
        expect(selectThinkingText(store.getState())).toBe('checking the inbox folder')
    })

    it('appends deltas to the running assistant message', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(assistantDelta('hel'))
        store.dispatch(assistantDelta('lo'))
        expect(selectMessages(store.getState())).toEqual([
            { role: 'user', content: 'hi' },
            { role: 'assistant', content: 'hello' }
        ])
        expect(selectThinking(store.getState())).toBe(false)
        expect(selectThinkingText(store.getState())).toBe('')
    })

    it('starts a fresh assistant message when the previous is not assistant', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(assistantDelta('hello'))
        store.dispatch(userMessage('again'))
        store.dispatch(assistantDelta('hi'))
        expect(selectMessages(store.getState())).toEqual([
            { role: 'user', content: 'hi' },
            { role: 'assistant', content: 'hello' },
            { role: 'user', content: 'again' },
            { role: 'assistant', content: 'hi' }
        ])
    })

    it('tracks the running tool and clears it on turn_done', () => {
        store.dispatch(toolActivity('meetings/standup.md'))
        expect(selectLastTool(store.getState())).toBe('meetings/standup.md')
        store.dispatch(turnDone({ conversationId: null }))
        expect(selectLastTool(store.getState())).toBeNull()
    })

    it('records the conversation id from turn_done', () => {
        store.dispatch(turnDone({ conversationId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1' }))
        expect(selectConversationId(store.getState())).toBe('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
        expect(selectStreaming(store.getState())).toBe(false)
    })

    it('keeps the conversation id when turn_done has none', () => {
        store.dispatch(turnDone({ conversationId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1' }))
        store.dispatch(turnDone({ conversationId: null }))
        expect(selectConversationId(store.getState())).toBe('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
    })

    it('records token usage from turn_done and clears it on a new send', () => {
        store.dispatch(
            turnDone({
                conversationId: null,
                usage: { input_tokens: 10, output_tokens: 4, cache_read_tokens: 5, cache_write_tokens: 3 }
            })
        )
        expect(selectLastUsage(store.getState())).toEqual({
            input_tokens: 10,
            output_tokens: 4,
            cache_read_tokens: 5,
            cache_write_tokens: 3
        })
        store.dispatch(userMessage('next question'))
        expect(selectLastUsage(store.getState())).toBeNull()
    })

    it('stays null when turn_done carries no usage', () => {
        store.dispatch(turnDone({ conversationId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1' }))
        expect(selectLastUsage(store.getState())).toBeNull()
    })

    it('surfaces error frames as a visible assistant message', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(chatError('turn cancelled'))
        expect(selectMessages(store.getState())).toEqual([
            { role: 'user', content: 'hi' },
            { role: 'assistant', content: '⚠️ turn cancelled' }
        ])
        expect(selectStreaming(store.getState())).toBe(false)
        expect(selectLastTool(store.getState())).toBeNull()
    })

    it('starts a fresh assistant message for a new turn after an error marker', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(chatError('cancelled'))
        store.dispatch(assistantStart())
        store.dispatch(assistantDelta('the answer'))
        expect(selectMessages(store.getState())).toEqual([
            { role: 'user', content: 'hi' },
            { role: 'assistant', content: '⚠️ cancelled' },
            { role: 'assistant', content: 'the answer' }
        ])
    })

    it('assistant_start between deltas opens a fresh message for the new turn', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(assistantDelta('first turn'))
        store.dispatch(assistantStart())
        store.dispatch(assistantDelta('second turn'))
        expect(selectMessages(store.getState())).toEqual([
            { role: 'user', content: 'hi' },
            { role: 'assistant', content: 'first turn' },
            { role: 'assistant', content: 'second turn' }
        ])
    })

    it('loads a conversation from history', () => {
        store.dispatch(userMessage('old'))
        store.dispatch(
            loadChat({
                messages: [{ role: 'user', content: 'from history' }],
                conversationId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1'
            })
        )
        expect(selectMessages(store.getState())).toEqual([{ role: 'user', content: 'from history' }])
        expect(selectConversationId(store.getState())).toBe('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
        expect(selectStreaming(store.getState())).toBe(false)
    })

    it('stops streaming on cancel', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(stopStreaming())
        expect(selectStreaming(store.getState())).toBe(false)
        expect(selectMessages(store.getState())).toEqual([{ role: 'user', content: 'hi' }])
    })

    it('resets everything for a new chat', () => {
        store.dispatch(userMessage('hi'))
        store.dispatch(turnDone({ conversationId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1' }))
        store.dispatch(assistantDelta('bye'))
        store.dispatch(resetChat())
        expect(selectMessages(store.getState())).toEqual([])
        expect(selectConversationId(store.getState())).toBeNull()
        expect(selectStreaming(store.getState())).toBe(false)
        expect(selectLastTool(store.getState())).toBeNull()
    })
})
