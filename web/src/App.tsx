import { Sidebar } from './components/Sidebar'
import { ChatPanel } from './components/ChatPanel'

export default function App() {
  return (
    <div className="flex h-screen bg-paper-50 text-ink-900 dark:bg-night-950 dark:text-paper-100 font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col min-w-0 border-l border-paper-200 dark:border-night-800">
        <ChatPanel />
      </main>
    </div>
  )
}
