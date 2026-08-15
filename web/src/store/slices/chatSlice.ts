import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { RootState } from '../index'

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
}

const initialState: ChatState = {
    messages: [],
    streaming: false,
    conversationId: null,
    lastTool: null,
    thinking: false,
    thinkingText: ''
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
        },
        assistantStart: (s) => {
            s.streaming = true
            s.thinking = true
        },
        assistantThinking: (s, a: PayloadAction<string>) => {
            s.thinking = true
            s.thinkingText = a.payload
        },
        assistantDelta: (s, a: PayloadAction<string>) => {
            s.thinking = false
            s.thinkingText = ''
            const last = s.messages[s.messages.length - 1]
            if (last && last.role === 'assistant') {
                last.content += a.payload
            } else {
                s.messages.push({ role: 'assistant', content: a.payload })
            }
        },
        toolActivity: (s, a: PayloadAction<string>) => {
            s.thinking = false
            s.thinkingText = ''
            s.lastTool = a.payload
        },
        turnDone: (s, a: PayloadAction<string | null>) => {
            if (a.payload !== null) s.conversationId = a.payload
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
        },
        chatError: (s, a: PayloadAction<string>) => {
            s.messages.push({ role: 'assistant', content: `⚠️ ${a.payload}` })
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
        },
        stopStreaming: (s) => {
            s.streaming = false
        },
        loadChat: (s, a: PayloadAction<{ messages: ChatMessage[]; conversationId: string }>) => {
            s.messages = a.payload.messages
            s.conversationId = a.payload.conversationId
            s.streaming = false
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
        },
        resetChat: (s) => {
            s.messages = []
            s.streaming = false
            s.conversationId = null
            s.lastTool = null
            s.thinking = false
            s.thinkingText = ''
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
