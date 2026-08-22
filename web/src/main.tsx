import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Provider } from 'react-redux'
import { App as AntdApp, ConfigProvider } from 'antd'
import './index.css'
import App from './app/App.tsx'
import { makeStore, fetchHealth, markNotificationRead, notify } from './store'
import { antdTheme } from './theme.ts'

const store = makeStore()
void store.dispatch(fetchHealth())

// REVIEW-AID: dummy notifications for the shell review — remove when the
// real sources wire in (tree_changed #10, rulebook #11, sync status #18,
// doctor warnings #20).
store.dispatch(notify({ kind: 'note', title: 'Note saved', body: 'knowledge/renovate-github-action.md' }))
store.dispatch(
    notify({ kind: 'rulebook', title: 'Rulebook updated', body: 'Wiki CLAUDE.md changed — applies next turn' })
)
store.dispatch(notify({ kind: 'sync', title: 'Git synced', body: '3 notes pushed to origin/main' }))
store.dispatch(
    notify({ kind: 'doctor', title: 'Model not configured', body: 'Using the CLI default until one is set' })
)
store.dispatch(notify({ kind: 'chat', title: 'Turn finished', body: 'The wiki search completed while you were away' }))
store.dispatch(markNotificationRead(store.getState().notifications.items[0]!.id))

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
