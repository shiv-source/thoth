import { render } from '@testing-library/react'
import { App as AntdApp } from 'antd'
import type { ReactElement } from 'react'
import { Provider } from 'react-redux'
import { makeStore } from '../store'

// Renders a component inside a fresh store + Provider — required by the
// typed react-redux hooks. A new store per call keeps tests isolated.
// The antd App wrapper mirrors main.tsx so App.useApp() (message,
// notification) renders for real in tests.
export function renderWithStore(ui: ReactElement) {
    const store = makeStore()
    return {
        store,
        ...render(
            <AntdApp>
                <Provider store={store}>{ui}</Provider>
            </AntdApp>
        )
    }
}
