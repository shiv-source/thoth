import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { RootState } from '../index'
import type { TokenUsage } from '../../ws/protocol'

export interface ChatMessage {
    role: 'user' | 'assistant'
    content: string
}

interface ChatState {
    messages: ChatMessage[]
    streaming: boolean
    conversationId: string | null
    lastTool: string | null
    thinking: boolean
    thinkingText: string
    // lastUsage is the last completed turn's token breakdown, rendered under
    // the final assistant message; null when the turn reported none.
    lastUsage: TokenUsage | null
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
    lastUsage: null,
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
            s.lastUsage = null
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
        turnDone: (s, a: PayloadAction<{ conversationId: string | null; usage?: TokenUsage }>) => {
            if (a.payload.conversationId !== null) s.conversationId = a.payload.conversationId
            s.lastUsage = a.payload.usage ?? null
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
            s.lastUsage = null
            s.freshMessage = false
        },
        stopStreaming: (s) => {
            s.streaming = false
            s.lastUsage = null
            s.freshMessage = false
        },
        loadChat: (s, a: PayloadAction<{ messages: ChatMessage[]; conversationId: string; usage?: TokenUsage }>) => {
            s.messages = a.payload.messages
            s.conversationId = a.payload.conversationId
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
            s.lastUsage = a.payload.usage ?? null
            s.freshMessage = false
        },
        resetChat: (s) => {
            s.messages = []
            s.streaming = false
            s.conversationId = null
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
            s.lastUsage = null
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
export const selectLastUsage = (s: RootState) => s.chat.lastUsage
