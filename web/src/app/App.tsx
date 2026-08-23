import { lazy, Suspense } from 'react'
import { Layout, Spin } from 'antd'
import { AppSider } from '../components/layout/AppSider'
import { Sidebar } from '../components/layout/Sidebar'
import { ChatPage } from '../pages/chat/ChatPage'
import { NotificationToasts } from '../shared/NotificationToasts'
import { DevBanner } from '../components/layout/DevBanner'

// Heavy views load on demand — Dashboard pulls chart.js, Settings and Notes
// pull their own trees. Chat (the default view) stays eager for first paint.
const NotesPage = lazy(() => import('../pages/notes/NotesPage').then((m) => ({ default: m.NotesPage })))
const DashboardPage = lazy(() => import('../pages/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage })))
const SearchPage = lazy(() => import('../pages/search/SearchPage').then((m) => ({ default: m.SearchPage })))
const SettingsPage = lazy(() => import('../pages/settings/SettingsPage').then((m) => ({ default: m.SettingsPage })))
const SetupPage = lazy(() => import('../pages/setup/SetupPage').then((m) => ({ default: m.SetupPage })))
import { navigateNote, navigateView, useViewRoute } from '../hooks/useView'
import { useViewShortcuts } from '../hooks/useViewShortcuts'
import { fetchHealth } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { selectHealth, selectHealthLoading } from '../store'

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
                        ) : health?.backend.api_key_configured || view === 'settings' ? (
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
                                {view === 'chat' && (
                                    <ChatPage onOpenSettings={openSettings} onOpenNote={openNoteHere} />
                                )}
                                {view === 'notes' && (
                                    <NotesPage
                                        openPath={segment}
                                        onOpenNote={openNoteHere}
                                        onOpenSettings={openSettings}
                                    />
                                )}
                                {view === 'dashboard' && <DashboardPage onOpenSettings={openSettings} />}
                                {view === 'search' && (
                                    <SearchPage onOpenNote={openNoteHere} onOpenSettings={openSettings} />
                                )}
                                {view === 'settings' && <SettingsPage />}
                            </Suspense>
                        ) : (
                            <Suspense fallback={<Spin size="large" />}>
                                <SetupPage health={health} loading={loading} onRecheck={() => void recheck()} />
                            </Suspense>
                        )}
                        <NotificationToasts />
                    </Layout.Content>
                </Layout>
            </Layout>
        </Layout>
    )
}
