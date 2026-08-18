import { Layout, Spin } from 'antd'
import { AppSider } from './components/AppSider'
import { Sidebar } from './components/Sidebar'
import { ChatPanel } from './components/ChatPanel'
import { NotesView } from './components/NotesView'
import { DashboardView } from './components/DashboardView'
import { SearchView } from './components/SearchView'
import { SettingsView } from './components/SettingsView'
import { SetupScreen } from './components/SetupScreen'
import { NotificationToasts } from './components/NotificationToasts'
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
            <AppSider />
            <Layout hasSider>
                {view === 'chat' && <Sidebar />}
                <Layout.Content className="relative flex min-w-0 flex-1 flex-col">
                    {loading && !health ? (
                        <div className="flex flex-1 items-center justify-center" role="status" aria-label="Loading">
                            <Spin size="large" />
                        </div>
                    ) : health?.claude.found ? (
                        <>
                            {view === 'chat' && <ChatPanel onOpenSettings={openSettings} />}
                            {view === 'notes' && (
                                <NotesView openPath={segment} onOpenNote={openNoteHere} onOpenSettings={openSettings} />
                            )}
                            {view === 'dashboard' && <DashboardView onOpenSettings={openSettings} />}
                            {view === 'search' && (
                                <SearchView onOpenNote={openNoteHere} onOpenSettings={openSettings} />
                            )}
                            {view === 'settings' && <SettingsView />}
                        </>
                    ) : (
                        <SetupScreen health={health} loading={loading} onRecheck={() => void recheck()} />
                    )}
                    <NotificationToasts />
                </Layout.Content>
            </Layout>
        </Layout>
    )
}
