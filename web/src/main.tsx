import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Provider } from 'react-redux'
import { App as AntdApp, ConfigProvider } from 'antd'
import './index.css'
import App from './app/App.tsx'
import { makeStore, fetchHealth } from './store'
import { antdTheme } from './theme.tsx'

const store = makeStore()
void store.dispatch(fetchHealth())

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <ConfigProvider theme={antdTheme}>
            <AntdApp>
                <Provider store={store}>
                    <App />
                </Provider>
            </AntdApp>
        </ConfigProvider>
    </StrictMode>
)
