import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { RootState } from '../index'

// ui holds screen-spanning chrome state: which overlays are open, tree
// expansion, keyboard selection. High-frequency local state (text drafts,
// per-keystroke input) deliberately stays in components — dispatching on
// every keystroke would re-render the whole tree for no benefit.
interface UiState {
    notificationsOpen: boolean
    notesExpandedKeys: string[]
    searchActive: number
    gitReposOpen: boolean
}

const initialState: UiState = {
    notificationsOpen: false,
    notesExpandedKeys: [],
    searchActive: -1,
    gitReposOpen: false
}

export const uiSlice = createSlice({
    name: 'ui',
    initialState,
    reducers: {
        setNotificationsOpen(s, a: PayloadAction<boolean>) {
            s.notificationsOpen = a.payload
        },
        setNotesExpandedKeys(s, a: PayloadAction<string[]>) {
            s.notesExpandedKeys = a.payload
        },
        setSearchActive(s, a: PayloadAction<number>) {
            s.searchActive = a.payload
        },
        setGitReposOpen(s, a: PayloadAction<boolean>) {
            s.gitReposOpen = a.payload
        }
    }
})

export const { setNotificationsOpen, setNotesExpandedKeys, setSearchActive, setGitReposOpen } = uiSlice.actions

export const selectNotificationsOpen = (s: RootState) => s.ui.notificationsOpen
export const selectNotesExpandedKeys = (s: RootState) => s.ui.notesExpandedKeys
export const selectSearchActive = (s: RootState) => s.ui.searchActive
export const selectGitReposOpen = (s: RootState) => s.ui.gitReposOpen
