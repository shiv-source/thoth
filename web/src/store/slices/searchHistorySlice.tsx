import { createSlice, type Middleware, type PayloadAction } from '@reduxjs/toolkit'
import type { RootState } from '../index'

// Search history is UI state persisted to localStorage so it survives
// reloads. The slice loads the initial items at store creation and the
// persistence middleware (wired in store/index.ts) writes them back on
// every commit/clear — only the search page reads them.
export const SEARCH_HISTORY_KEY = 'thoth.searchHistory'
export const SEARCH_HISTORY_MAX = 10

interface SearchHistoryState {
    items: string[]
}

function loadHistory(): string[] {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(SEARCH_HISTORY_KEY) ?? '[]')
        return Array.isArray(parsed) ? parsed.filter((x): x is string => typeof x === 'string') : []
    } catch {
        return []
    }
}

export const searchHistorySlice = createSlice({
    name: 'searchHistory',
    // Lazy initializer — the history is read per store creation, not at
    // module load (tests and hot reloads seed localStorage later).
    initialState: (): SearchHistoryState => ({ items: loadHistory() }),
    reducers: {
        // commitSearch records a deliberate search — deduped, most-recent
        // first, capped so the store cannot grow unbounded.
        commitSearch: (s, a: PayloadAction<string>) => {
            const q = a.payload.trim()
            if (!q) return
            s.items = [q, ...s.items.filter((x) => x !== q)].slice(0, SEARCH_HISTORY_MAX)
        },
        clearSearchHistory: (s) => {
            s.items = []
        }
    }
})

export const { commitSearch, clearSearchHistory } = searchHistorySlice.actions

export const selectSearchHistory = (s: RootState) => s.searchHistory.items

// persistSearchHistory writes the slice back to localStorage on every
// commit/clear — reducers stay pure, the side effect lives here. The state
// is typed structurally (not RootState) because RootState's definition
// includes this middleware — annotating with it would be a type cycle.
interface SearchHistoryStoreShape {
    searchHistory: { items: string[] }
}

export const persistSearchHistory: Middleware<unknown, SearchHistoryStoreShape> = (store) => (next) => (action) => {
    const result = next(action)
    if (commitSearch.match(action) || clearSearchHistory.match(action)) {
        localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(store.getState().searchHistory.items))
    }
    return result
}
