import { useEffect, useRef, useState } from 'react'
import { Bell, Settings } from 'lucide-react'
import { IconButton } from './IconButton'
import { selectUnreadCount } from '../store'
import { useAppSelector } from '../store/hooks'
import { NotificationPanel } from './NotificationPanel'
import { Tooltip } from './Tooltip'

export function TopBar({ title, onOpenSettings }: { title: string; onOpenSettings?: () => void }) {
    const unread = useAppSelector(selectUnreadCount)
    const [panelOpen, setPanelOpen] = useState(false)
    const headerRef = useRef<HTMLElement>(null)

    // While the panel is open: Escape closes it, and a press anywhere
    // outside the header (which owns both the bell and the panel) closes it.
    useEffect(() => {
        if (!panelOpen) return
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') setPanelOpen(false)
        }
        const onDown = (e: MouseEvent) => {
            if (headerRef.current && !headerRef.current.contains(e.target as Node)) setPanelOpen(false)
        }
        document.addEventListener('keydown', onKey)
        document.addEventListener('mousedown', onDown)
        return () => {
            document.removeEventListener('keydown', onKey)
            document.removeEventListener('mousedown', onDown)
        }
    }, [panelOpen])

    return (
        <header
            ref={headerRef}
            className="relative flex h-14 shrink-0 items-center justify-between gap-4 border-b border-line bg-surface px-4"
        >
            <h1 className="truncate text-sm font-medium text-ink">{title}</h1>
            <div className="flex shrink-0 items-center gap-1">
                <span className="relative">
                    <IconButton label="Notifications" onClick={() => setPanelOpen((o) => !o)}>
                        <Bell className="h-4 w-4" aria-hidden="true" />
                    </IconButton>
                    {unread > 0 && (
                        <span
                            aria-label={`${unread} unread`}
                            className="absolute right-0.5 top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-accent px-1 text-[10px] font-medium text-accent-ink"
                        >
                            {unread}
                        </span>
                    )}
                </span>
                {onOpenSettings && (
                    <Tooltip label="Settings">
                        <IconButton label="Settings" onClick={onOpenSettings}>
                            <Settings className="h-4 w-4" aria-hidden="true" />
                        </IconButton>
                    </Tooltip>
                )}
            </div>
            {panelOpen && <NotificationPanel onClose={() => setPanelOpen(false)} />}
        </header>
    )
}
