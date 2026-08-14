import { useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { ChatPanel } from './components/ChatPanel'
import { NoteViewer } from './components/NoteViewer'

export default function App() {
  const [openNote, setOpenNote] = useState<string | null>(null)

  return (
    <div className="flex h-screen bg-app font-sans text-ink">
      <Sidebar openPath={openNote} onOpenNote={setOpenNote} />
      <main className="flex min-w-0 flex-1 flex-col">
        <ChatPanel />
      </main>
      {openNote && <NoteViewer path={openNote} onClose={() => setOpenNote(null)} />}
    </div>
  )
}
