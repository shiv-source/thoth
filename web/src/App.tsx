import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { ChatPanel } from './components/ChatPanel'
import { NavRail } from './components/NavRail'
import { NotesView } from './components/NotesView'
import { DashboardView } from './components/DashboardView'
import { SearchView } from './components/SearchView'
import { NoteViewer } from './components/NoteViewer'
import { SetupScreen } from './components/SetupScreen'
import { SettingsModal } from './components/SettingsModal'
import { ToastProvider } from './components/Toast'
import { useView } from './hooks/useView'
import { fetchHealth } from './store'
import { useAppDispatch, useAppSelector } from './store/hooks'
import { selectHealth, selectHealthLoading } from './store'

export default function App() {
    const view = useView()
    const [openNote, setOpenNote] = useState<string | null>(null)
    const [settingsOpen, setSettingsOpen] = useState(false)
    const dispatch = useAppDispatch()
    const health = useAppSelector(selectHealth)
    const loading = useAppSelector(selectHealthLoading)
    const recheck = () => void dispatch(fetchHealth())
    const openSettings = () => setSettingsOpen(true)

    return (
        <ToastProvider>
            <div className="flex h-screen bg-app font-sans text-ink">
                <NavRail />
                <Sidebar health={health} loading={loading} />
                <main className="flex min-w-0 flex-1 flex-col">
                    {loading && !health ? (
                        <div className="flex flex-1 items-center justify-center" role="status" aria-label="Loading">
                            <span
                                aria-hidden="true"
                                className="h-8 w-8 animate-spin rounded-full border-2 border-line border-t-accent"
                            />
                        </div>
                    ) : health?.claude.found ? (
                        <>
                            {view === 'chat' && <ChatPanel onOpenSettings={openSettings} />}
                            {view === 'notes' && (
                                <NotesView openPath={openNote} onOpenNote={setOpenNote} onOpenSettings={openSettings} />
                            )}
                            {view === 'dashboard' && <DashboardView onOpenSettings={openSettings} />}
                            {view === 'search' && <SearchView onOpenNote={setOpenNote} onOpenSettings={openSettings} />}
                        </>
                    ) : (
                        <SetupScreen health={health} loading={loading} onRecheck={() => void recheck()} />
                    )}
                </main>
                {view !== 'notes' && openNote && <NoteViewer path={openNote} onClose={() => setOpenNote(null)} />}
                {settingsOpen && <SettingsModal onClose={() => setSettingsOpen(false)} />}
            </div>
        </ToastProvider>
    )
}
