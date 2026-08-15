import { configureStore } from '@reduxjs/toolkit'
import { healthSlice } from './slices/healthSlice'
import { settingsSlice } from './slices/settingsSlice'
import { conversationsSlice } from './slices/conversationsSlice'
import { chatSlice } from './slices/chatSlice'
import { connectionSlice } from './slices/connectionSlice'

export function makeStore() {
    return configureStore({
        reducer: {
            health: healthSlice.reducer,
            settings: settingsSlice.reducer,
            conversations: conversationsSlice.reducer,
            chat: chatSlice.reducer,
            connection: connectionSlice.reducer
        }
    })
}

export type AppStore = ReturnType<typeof makeStore>
export type RootState = ReturnType<AppStore['getState']>
export type AppDispatch = AppStore['dispatch']

export { fetchHealth, selectHealth, selectHealthLoading } from './slices/healthSlice'
export { fetchSettings, saveSettings, selectSettings } from './slices/settingsSlice'
export {
    fetchConversations,
    deleteConversation,
    selectConversations,
    selectConversationsLoading
} from './slices/conversationsSlice'
export {
    userMessage,
    assistantStart,
    assistantThinking,
    assistantDelta,
    toolActivity,
    turnDone,
    chatError,
    stopStreaming,
    loadChat,
    resetChat,
    selectMessages,
    selectStreaming,
    selectConversationId,
    selectLastTool,
    selectThinking,
    selectThinkingText
} from './slices/chatSlice'
export { setStatus, selectConnectionStatus } from './slices/connectionSlice'
