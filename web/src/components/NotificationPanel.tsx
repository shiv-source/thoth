import { BellOff, X } from 'lucide-react'
import { dismissNotification, markAllRead, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { EmptyState } from './EmptyState'
import { IconButton } from './IconButton'
import { NotificationIcon } from './notifications'

export function NotificationPanel({ onClose }: { onClose: () => void }) {
    const items = useAppSelector(selectNotifications)
    const dispatch = useAppDispatch()

    return (
        <div
            role="dialog"
            aria-label="Notifications"
            className="absolute right-2 top-full z-40 mt-1 flex max-h-96 w-80 flex-col rounded-xl border border-line bg-surface shadow-lg"
        >
            <header className="flex items-center justify-between border-b border-line px-3 py-2">
                <span className="text-sm font-medium text-ink">Notifications</span>
                <div className="flex items-center gap-1">
                    <IconButton label="Mark all as read" onClick={() => dispatch(markAllRead())}>
                        <BellOff className="h-3.5 w-3.5" aria-hidden="true" />
                    </IconButton>
                    <IconButton label="Close notifications" onClick={onClose}>
                        <X className="h-3.5 w-3.5" aria-hidden="true" />
                    </IconButton>
                </div>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto">
                {items.length === 0 ? (
                    <EmptyState icon="🔔" title="No notifications yet" className="py-6 text-sm" />
                ) : (
                    <ul className="divide-y divide-line">
                        {items.map((n) => (
                            <li key={n.id} className="group relative px-3 py-2.5">
                                <span className="mr-2">
                                    <NotificationIcon kind={n.kind} />
                                </span>
                                <span className={`text-sm ${n.read ? 'text-subtle' : 'font-medium text-ink'}`}>
                                    {n.title}
                                </span>
                                {n.body && <p className="mt-0.5 pl-6 text-xs text-subtle">{n.body}</p>}
                                <span className="absolute right-2 top-2 hidden group-hover:block">
                                    <IconButton
                                        label={`Dismiss: ${n.title}`}
                                        onClick={() => dispatch(dismissNotification(n.id))}
                                    >
                                        <X className="h-3 w-3" aria-hidden="true" />
                                    </IconButton>
                                </span>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    )
}
