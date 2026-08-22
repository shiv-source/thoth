import { describe, expect, it } from 'vitest'
import { makeStore } from '../index'
import { selectConnectionStatus, setStatus } from './connectionSlice'

describe('connectionSlice', () => {
    it('starts connected', () => {
        expect(selectConnectionStatus(makeStore().getState())).toBe('connected')
    })

    it('tracks status changes', () => {
        const store = makeStore()
        store.dispatch(setStatus('reconnecting'))
        expect(selectConnectionStatus(store.getState())).toBe('reconnecting')
        store.dispatch(setStatus('disconnected'))
        expect(selectConnectionStatus(store.getState())).toBe('disconnected')
        store.dispatch(setStatus('connected'))
        expect(selectConnectionStatus(store.getState())).toBe('connected')
    })
})
