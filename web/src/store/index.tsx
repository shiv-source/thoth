import { configureStore } from '@reduxjs/toolkit'
import { healthSlice } from './slices/healthSlice'
import { settingsSlice } from './slices/settingsSlice'
import { conversationsSlice } from './slices/conversationsSlice'
import { chatSlice } from './slices/chatSlice'
import { connectionSlice } from './slices/connectionSlice'
import { notificationsSlice } from './slices/notificationsSlice'
import { persistSearchHistory, searchHistorySlice } from './slices/searchHistorySlice'
import { uiSlice } from './slices/uiSlice'
import { wikiSlice } from './slices/wikiSlice'
import { noteSlice } from './slices/noteSlice'
import { searchSlice } from './slices/searchSlice'
import { doctorSlice } from './slices/doctorSlice'
import { gitSlice } from './slices/gitSlice'

export function makeStore() {
    return configureStore({
        reducer: {
            health: healthSlice.reducer,
            settings: settingsSlice.reducer,
            conversations: conversationsSlice.reducer,
            chat: chatSlice.reducer,
            connection: connectionSlice.reducer,
            notifications: notificationsSlice.reducer,
            searchHistory: searchHistorySlice.reducer,
            ui: uiSlice.reducer,
            wiki: wikiSlice.reducer,
            note: noteSlice.reducer,
            search: searchSlice.reducer,
            doctor: doctorSlice.reducer,
            git: gitSlice.reducer
        },
        middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(persistSearchHistory)
    })
}

export type AppStore = ReturnType<typeof makeStore>
export type RootState = ReturnType<AppStore['getState']>
export type AppDispatch = AppStore['dispatch']

export { fetchHealth, selectHealth, selectHealthLoading } from './slices/healthSlice'
export {
    fetchSettings,
    saveSettings,
    fetchModels,
    createModel,
    updateModel,
    deleteModel,
    fetchProviders,
    createProvider,
    updateProvider,
    deleteProvider,
    selectSettings,
    selectModelGroups,
    selectModelList,
    selectProviders
} from './slices/settingsSlice'
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
    selectThinkingText,
    selectLastUsage
} from './slices/chatSlice'
export { setStatus, selectConnectionStatus } from './slices/connectionSlice'
export {
    notify,
    markNotificationRead,
    markAllRead,
    dismissNotification,
    selectNotifications,
    selectUnreadCount
} from './slices/notificationsSlice'
export { commitSearch, clearSearchHistory, selectSearchHistory } from './slices/searchHistorySlice'
export {
    setNotificationsOpen,
    setNotesExpandedKeys,
    setSearchActive,
    setGitReposOpen,
    selectNotificationsOpen,
    selectNotesExpandedKeys,
    selectSearchActive,
    selectGitReposOpen
} from './slices/uiSlice'
export { fetchTree, selectWikiNodes, selectWikiLoading, selectWikiError, collectTreeInfo } from './slices/wikiSlice'
export { fetchNote, selectNoteContent, selectNoteLoading, selectNoteError } from './slices/noteSlice'
export {
    searchNotes,
    clearSearch,
    selectSearchResults,
    selectSearchLoading,
    selectSearchError
} from './slices/searchSlice'
export { runDoctor, selectDoctorChecks, selectDoctorRunning, selectDoctorError } from './slices/doctorSlice'
export {
    fetchGitAuth,
    fetchGitRepos,
    connectGit,
    pushWiki,
    disconnectGit,
    selectGitAuth,
    selectGitRepos,
    selectGitLoading,
    selectGitConnecting,
    selectGitPushing,
    selectGitError
} from './slices/gitSlice'
