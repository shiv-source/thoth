import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { Provider } from 'react-redux'
import { makeStore } from '../store'

// Renders a component inside a fresh store + Provider — required by the
// typed react-redux hooks. A new store per call keeps tests isolated.
export function renderWithStore(ui: ReactElement) {
    const store = makeStore()
    return { store, ...render(<Provider store={store}>{ui}</Provider>) }
}
