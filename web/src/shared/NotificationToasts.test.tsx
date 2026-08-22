import { act, fireEvent, render, screen } from '@testing-library/react'
import { Provider } from 'react-redux'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeStore, notify, selectNotifications } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import { NotificationToasts } from './NotificationToasts'

describe('NotificationToasts', () => {
    beforeEach(() => {
        vi.useFakeTimers()
    })

    afterEach(() => {
        vi.useRealTimers()
    })

    it('does not toast notifications that already existed on mount', () => {
        // Seed before mount, as main.tsx does with the review dummy data.
        const store = makeStore()
        store.dispatch(notify({ kind: 'note', title: 'old' }))
        render(
            <Provider store={store}>
                <NotificationToasts />
            </Provider>
        )
        expect(screen.queryByText('old')).not.toBeInTheDocument()
    })

    it('toasts new notifications in the top-left corner', () => {
        const { store } = renderWithStore(<NotificationToasts />)
        act(() => {
            store.dispatch(notify({ kind: 'sync', title: 'Git synced', body: '3 notes pushed' }))
        })
        expect(screen.getByText('Git synced')).toBeInTheDocument()
        expect(screen.getByText('3 notes pushed')).toBeInTheDocument()
    })

    it('dismisses a toast on click', () => {
        const { store } = renderWithStore(<NotificationToasts />)
        act(() => {
            store.dispatch(notify({ kind: 'note', title: 'Note saved' }))
        })
        // fireEvent is synchronous — userEvent's internal timer waits would
        // trip the 5s auto-dismiss under fake timers.
        fireEvent.click(screen.getByRole('button', { name: 'Dismiss: Note saved' }))
        expect(screen.queryByText('Note saved')).not.toBeInTheDocument()
        expect(selectNotifications(store.getState())).toHaveLength(0)
    })

    it('auto-dismisses after five seconds', () => {
        const { store } = renderWithStore(<NotificationToasts />)
        act(() => {
            store.dispatch(notify({ kind: 'chat', title: 'Turn finished' }))
        })
        expect(screen.getByText('Turn finished')).toBeInTheDocument()
        act(() => {
            vi.advanceTimersByTime(5000)
        })
        expect(screen.queryByText('Turn finished')).not.toBeInTheDocument()
    })
})
