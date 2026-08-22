import { describe, expect, it } from 'vitest'
import { makeStore } from '../index'
import {
    selectGitReposOpen,
    selectNotificationsOpen,
    selectNotesExpandedKeys,
    selectSearchActive,
    setGitReposOpen,
    setNotificationsOpen,
    setNotesExpandedKeys,
    setSearchActive
} from '../index'

describe('uiSlice', () => {
    it('tracks chrome state through its actions', () => {
        const store = makeStore()

        store.dispatch(setNotificationsOpen(true))
        expect(selectNotificationsOpen(store.getState())).toBe(true)
        store.dispatch(setNotificationsOpen(false))
        expect(selectNotificationsOpen(store.getState())).toBe(false)

        store.dispatch(setNotesExpandedKeys(['a', 'b']))
        expect(selectNotesExpandedKeys(store.getState())).toEqual(['a', 'b'])

        store.dispatch(setSearchActive(2))
        expect(selectSearchActive(store.getState())).toBe(2)
        store.dispatch(setSearchActive(-1))
        expect(selectSearchActive(store.getState())).toBe(-1)

        store.dispatch(setGitReposOpen(true))
        expect(selectGitReposOpen(store.getState())).toBe(true)
    })
})
