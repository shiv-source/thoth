import { FileText, LayoutDashboard, MessageSquare, Search, Settings } from 'lucide-react'
import { navigateView, useView, type View } from '../hooks/useView'

const VIEWS: { view: View; label: string; icon: typeof MessageSquare }[] = [
    { view: 'chat', label: 'Chat', icon: MessageSquare },
    { view: 'notes', label: 'Notes', icon: FileText },
    { view: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { view: 'search', label: 'Search', icon: Search }
]

// NavRail is the app's persistent view switcher plus the notification bell.
// Views route through the URL hash (#/chat, #/notes, …), so back/forward
// and deep links keep working.
export function NavRail() {
    const view = useView()

    return (
        <nav
            aria-label="App views"
            className="relative flex w-14 shrink-0 flex-col items-center gap-1 border-r border-line bg-surface py-2"
        >
            {VIEWS.map(({ view: v, label, icon: Icon }) => (
                <button
                    key={v}
                    type="button"
                    onClick={() => navigateView(v)}
                    aria-label={label}
                    aria-current={view === v ? 'page' : undefined}
                    className={`flex h-10 w-10 items-center justify-center rounded-lg transition ${
                        view === v ? 'bg-accent text-accent-ink' : 'text-subtle hover:bg-raised hover:text-ink'
                    }`}
                >
                    <Icon className="h-5 w-5" aria-hidden="true" />
                </button>
            ))}
            <div className="mt-auto flex flex-col items-center">
                <button
                    type="button"
                    onClick={() => navigateView('settings')}
                    aria-label="Settings"
                    aria-current={view === 'settings' ? 'page' : undefined}
                    className={`flex h-10 w-10 items-center justify-center rounded-lg transition ${
                        view === 'settings' ? 'bg-accent text-accent-ink' : 'text-subtle hover:bg-raised hover:text-ink'
                    }`}
                >
                    <Settings className="h-5 w-5" aria-hidden="true" />
                </button>
            </div>
        </nav>
    )
}
