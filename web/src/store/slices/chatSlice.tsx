import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { RootState } from '../index'
import type { TokenUsage } from '../../ws/protocol'

export interface ChatMessage {
    role: 'user' | 'assistant'
    content: string
    // usage is the turn's token breakdown for this assistant message, set on
    // turn_done (live turns) or from persisted history (loadChat); absent when
    // the provider reported none.
    usage?: TokenUsage
    // durationSecs is how long the assistant turn took to generate this reply,
    // set on turn_done or from persisted history; absent when the turn was
    // instant (under a second) or predates duration tracking.
    durationSecs?: number
}

interface ChatState {
    messages: ChatMessage[]
    streaming: boolean
    conversationId: string | null
    lastTool: string | null
    thinking: boolean
    thinkingText: string
    // freshMessage marks that assistant_start opened a new turn whose first
    // delta must start a NEW assistant message (never append into a previous
    // turn's message or an error marker).
    freshMessage: boolean
}

const initialState: ChatState = {
    messages: [],
    streaming: false,
    conversationId: null,
    lastTool: null,
    thinking: false,
    thinkingText: '',
    freshMessage: false
}

// The chat slice is a pure state machine over WS frames: useChat dispatches
// these actions (and calls into the socket for the wire side), so the full
// conversation state lives in the store and survives component remounts.
export const chatSlice = createSlice({
    name: 'chat',
    initialState,
    reducers: {
        userMessage: (s, a: PayloadAction<string>) => {
            s.messages.push({ role: 'user', content: a.payload })
            s.streaming = true
            s.freshMessage = false
        },
        assistantStart: (s) => {
            s.streaming = true
            s.thinking = true
            s.freshMessage = true
        },
        assistantThinking: (s, a: PayloadAction<string>) => {
            s.thinking = true
            s.thinkingText = a.payload
        },
        assistantDelta: (s, a: PayloadAction<string>) => {
            s.thinking = false
            s.thinkingText = ''
            const last = s.messages[s.messages.length - 1]
            if (!s.freshMessage && last && last.role === 'assistant') {
                last.content += a.payload
            } else {
                s.messages.push({ role: 'assistant', content: a.payload })
            }
            s.freshMessage = false
        },
        toolActivity: (s, a: PayloadAction<string>) => {
            s.thinking = false
            s.thinkingText = ''
            s.lastTool = a.payload
        },
        turnDone: (
            s,
            a: PayloadAction<{ conversationId: string | null; usage?: TokenUsage; durationSecs?: number }>
        ) => {
            if (a.payload.conversationId !== null) s.conversationId = a.payload.conversationId
            // Usage and duration ride on the assistant message that just
            // finished; a tool-only turn stores no text message, so there is
            // nowhere to attach them and they are dropped.
            const last = s.messages[s.messages.length - 1]
            if (last && last.role === 'assistant') {
                if (a.payload.usage) last.usage = a.payload.usage
                if (a.payload.durationSecs) last.durationSecs = a.payload.durationSecs
            }
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
            s.freshMessage = false
        },
        chatError: (s, a: PayloadAction<string>) => {
            s.messages.push({ role: 'assistant', content: `⚠️ ${a.payload}` })
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
            s.freshMessage = false
        },
        stopStreaming: (s) => {
            s.streaming = false
            s.freshMessage = false
        },
        loadChat: (s, a: PayloadAction<{ messages: ChatMessage[]; conversationId: string }>) => {
            s.messages = a.payload.messages
            s.conversationId = a.payload.conversationId
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
            s.freshMessage = false
        },
        resetChat: (s) => {
            s.messages = []
            s.streaming = false
            s.conversationId = null
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
            s.freshMessage = false
        }
    }
})

export const {
    userMessage,
    assistantStart,
    assistantThinking,
    assistantDelta,
    toolActivity,
    turnDone,
    chatError,
    stopStreaming,
    loadChat,
    resetChat
} = chatSlice.actions

export const selectMessages = (s: RootState) => s.chat.messages
export const selectStreaming = (s: RootState) => s.chat.streaming
export const selectConversationId = (s: RootState) => s.chat.conversationId
export const selectLastTool = (s: RootState) => s.chat.lastTool
export const selectThinking = (s: RootState) => s.chat.thinking
export const selectThinkingText = (s: RootState) => s.chat.thinkingText
// selectTotalUsage sums the per-turn token breakdowns across every assistant
// message in the conversation. All counters zero when no turn reported usage.
export const selectTotalUsage = (s: RootState): TokenUsage => {
    const total: TokenUsage = { input_tokens: 0, output_tokens: 0, cache_read_tokens: 0, cache_write_tokens: 0 }
    for (const m of s.chat.messages) {
        if (m.role !== 'assistant' || !m.usage) continue
        total.input_tokens += m.usage.input_tokens
        total.output_tokens += m.usage.output_tokens
        total.cache_read_tokens += m.usage.cache_read_tokens
        total.cache_write_tokens += m.usage.cache_write_tokens
    }
    return total
}
