import { useState } from 'react'
import { Bell, FileText, LayoutDashboard, MessageSquare, Search } from 'lucide-react'
import { navigateView, useView, type View } from '../hooks/useView'
import { selectUnreadCount } from '../store'
import { useAppSelector } from '../store/hooks'
import { NotificationPanel } from './NotificationPanel'

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
    const unread = useAppSelector(selectUnreadCount)
    const [panelOpen, setPanelOpen] = useState(false)

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
                    onClick={() => setPanelOpen((o) => !o)}
                    aria-label="Notifications"
                    aria-expanded={panelOpen}
                    className="relative flex h-10 w-10 items-center justify-center rounded-lg text-subtle transition hover:bg-raised hover:text-ink"
                >
                    <Bell className="h-5 w-5" aria-hidden="true" />
                    {unread > 0 && (
                        <span
                            aria-label={`${unread} unread`}
                            className="absolute right-0.5 top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-accent px-1 text-[10px] font-medium text-accent-ink"
                        >
                            {unread}
                        </span>
                    )}
                </button>
            </div>
            {panelOpen && <NotificationPanel onClose={() => setPanelOpen(false)} />}
        </nav>
    )
}
