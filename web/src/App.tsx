import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { ChatPanel } from './components/ChatPanel'
import { NoteViewer } from './components/NoteViewer'
import { SetupScreen } from './components/SetupScreen'
import { ToastProvider } from './components/Toast'
import { fetchHealth } from './store'
import { useAppDispatch, useAppSelector } from './store/hooks'
import { selectHealth, selectHealthLoading } from './store'

export default function App() {
    const [openNote, setOpenNote] = useState<string | null>(null)
    const dispatch = useAppDispatch()
    const health = useAppSelector(selectHealth)
    const loading = useAppSelector(selectHealthLoading)
    const recheck = () => void dispatch(fetchHealth())

    return (
        <ToastProvider>
            <div className="flex h-screen bg-app font-sans text-ink">
                <Sidebar openPath={openNote} onOpenNote={setOpenNote} health={health} loading={loading} />
                <main className="flex min-w-0 flex-1 flex-col">
                    {loading && !health ? (
                        <div className="flex flex-1 items-center justify-center" role="status" aria-label="Loading">
                            <span
                                aria-hidden="true"
                                className="h-8 w-8 animate-spin rounded-full border-2 border-line border-t-accent"
                            />
                        </div>
                    ) : health?.claude.found ? (
                        <ChatPanel />
                    ) : (
                        <SetupScreen health={health} loading={loading} onRecheck={() => void recheck()} />
                    )}
                </main>
                {openNote && <NoteViewer path={openNote} onClose={() => setOpenNote(null)} />}
            </div>
        </ToastProvider>
    )
}
