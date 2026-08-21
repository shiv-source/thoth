import { lazy, Suspense } from 'react'
import { Layout, Spin } from 'antd'
import { AppSider } from './components/AppSider'
import { Sidebar } from './components/Sidebar'
import { ChatPanel } from './components/ChatPanel'
import { NotificationToasts } from './components/NotificationToasts'
import { DevBanner } from './components/DevBanner'

// Heavy views load on demand — Dashboard pulls chart.js, Settings and Notes
// pull their own trees. Chat (the default view) stays eager for first paint.
const NotesView = lazy(() => import('./components/NotesView').then((m) => ({ default: m.NotesView })))
const DashboardView = lazy(() => import('./components/DashboardView').then((m) => ({ default: m.DashboardView })))
const SearchView = lazy(() => import('./components/SearchView').then((m) => ({ default: m.SearchView })))
const SettingsView = lazy(() => import('./components/SettingsView').then((m) => ({ default: m.SettingsView })))
const SetupScreen = lazy(() => import('./components/SetupScreen').then((m) => ({ default: m.SetupScreen })))
import { navigateNote, navigateView, useViewRoute } from './hooks/useView'
import { useViewShortcuts } from './hooks/useViewShortcuts'
import { fetchHealth } from './store'
import { useAppDispatch, useAppSelector } from './store/hooks'
import { selectHealth, selectHealthLoading } from './store'

export default function App() {
    const { view, segment } = useViewRoute()
    useViewShortcuts()
    const dispatch = useAppDispatch()
    const health = useAppSelector(selectHealth)
    const loading = useAppSelector(selectHealthLoading)
    const recheck = () => void dispatch(fetchHealth())
    const openSettings = () => navigateView('settings')
    // Notes open only in the Notes view's inline reader — any view that
    // opens one routes there, and the path rides the URL hash so the open
    // note survives a reload.
    const openNoteHere = (path: string | null) => navigateNote(path)

    return (
        <Layout className="h-screen bg-app font-sans text-ink">
            <DevBanner dev={health?.dev ?? false} commit={health?.commit ?? ''} />
            <Layout hasSider>
                <AppSider />
                <Layout hasSider>
                    {view === 'chat' && <Sidebar />}
                    <Layout.Content className="relative flex min-w-0 flex-1 flex-col">
                        {loading && !health ? (
                            <div className="flex flex-1 items-center justify-center" role="status" aria-label="Loading">
                                <Spin size="large" />
                            </div>
                        ) : health?.backend.api_key_configured ? (
                            <Suspense
                                fallback={
                                    <div
                                        className="flex flex-1 items-center justify-center"
                                        role="status"
                                        aria-label="Loading"
                                    >
                                        <Spin size="large" />
                                    </div>
                                }
                            >
                                {view === 'chat' && <ChatPanel onOpenSettings={openSettings} />}
                                {view === 'notes' && (
                                    <NotesView
                                        openPath={segment}
                                        onOpenNote={openNoteHere}
                                        onOpenSettings={openSettings}
                                    />
                                )}
                                {view === 'dashboard' && <DashboardView onOpenSettings={openSettings} />}
                                {view === 'search' && (
                                    <SearchView onOpenNote={openNoteHere} onOpenSettings={openSettings} />
                                )}
                                {view === 'settings' && <SettingsView />}
                            </Suspense>
                        ) : (
                            <Suspense fallback={<Spin size="large" />}>
                                <SetupScreen health={health} loading={loading} onRecheck={() => void recheck()} />
                            </Suspense>
                        )}
                        <NotificationToasts />
                    </Layout.Content>
                </Layout>
            </Layout>
        </Layout>
    )
}
