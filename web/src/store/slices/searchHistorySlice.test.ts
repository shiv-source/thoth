import { beforeEach, describe, expect, it } from 'vitest'
import { makeStore, clearSearchHistory, commitSearch, selectSearchHistory } from '../index'
import { SEARCH_HISTORY_KEY, SEARCH_HISTORY_MAX } from './searchHistorySlice'

describe('searchHistorySlice', () => {
    beforeEach(() => {
        localStorage.clear()
    })

    it('loads the initial items from localStorage', () => {
        localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(['deploy', 'bookmarks']))
        const store = makeStore()
        expect(selectSearchHistory(store.getState())).toEqual(['deploy', 'bookmarks'])
    })

    it('ignores a malformed localStorage payload', () => {
        localStorage.setItem(SEARCH_HISTORY_KEY, '{not json')
        const store = makeStore()
        expect(selectSearchHistory(store.getState())).toEqual([])
    })

    it('commits a search to the front, deduped and trimmed, and persists it', () => {
        const store = makeStore()
        store.dispatch(commitSearch('bookmarks'))
        store.dispatch(commitSearch('  deploy  '))
        store.dispatch(commitSearch('bookmarks'))

        expect(selectSearchHistory(store.getState())).toEqual(['bookmarks', 'deploy'])
        expect(JSON.parse(localStorage.getItem(SEARCH_HISTORY_KEY) ?? '[]')).toEqual(['bookmarks', 'deploy'])
    })

    it('ignores blank commits', () => {
        const store = makeStore()
        store.dispatch(commitSearch('   '))
        expect(selectSearchHistory(store.getState())).toEqual([])
    })

    it('caps the history so the store cannot grow unbounded', () => {
        const store = makeStore()
        for (let i = 0; i < SEARCH_HISTORY_MAX + 3; i++) store.dispatch(commitSearch(`q${i}`))
        const items = selectSearchHistory(store.getState())
        expect(items.length).toBe(SEARCH_HISTORY_MAX)
        expect(items[0]).toBe(`q${SEARCH_HISTORY_MAX + 2}`)
    })

    it('clears the history and persists the empty list', () => {
        localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(['deploy']))
        const store = makeStore()
        store.dispatch(clearSearchHistory())
        expect(selectSearchHistory(store.getState())).toEqual([])
        expect(JSON.parse(localStorage.getItem(SEARCH_HISTORY_KEY) ?? 'null')).toEqual([])
    })
})
