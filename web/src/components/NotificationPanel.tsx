import { BellOff, X } from 'lucide-react'
import { dismissNotification, markAllRead, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import type { NotificationKind } from '../store/slices/notificationsSlice'

const KIND_ICON: Record<NotificationKind, string> = {
    sync: '🔄',
    note: '📝',
    rulebook: '📜',
    chat: '💬',
    doctor: '🩺'
}

export function NotificationPanel({ onClose }: { onClose: () => void }) {
    const items = useAppSelector(selectNotifications)
    const dispatch = useAppDispatch()

    return (
        <div
            role="dialog"
            aria-label="Notifications"
            className="absolute left-full top-0 z-40 ml-1 flex max-h-96 w-80 flex-col rounded-xl border border-line bg-surface shadow-lg"
        >
            <header className="flex items-center justify-between border-b border-line px-3 py-2">
                <span className="text-sm font-medium text-ink">Notifications</span>
                <div className="flex items-center gap-1">
                    <button
                        type="button"
                        onClick={() => dispatch(markAllRead())}
                        aria-label="Mark all as read"
                        className="rounded-md p-1.5 text-subtle transition hover:bg-raised hover:text-ink"
                    >
                        <BellOff className="h-3.5 w-3.5" aria-hidden="true" />
                    </button>
                    <button
                        type="button"
                        onClick={onClose}
                        aria-label="Close notifications"
                        className="rounded-md p-1.5 text-subtle transition hover:bg-raised hover:text-ink"
                    >
                        <X className="h-3.5 w-3.5" aria-hidden="true" />
                    </button>
                </div>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto">
                {items.length === 0 ? (
                    <p className="px-4 py-6 text-center text-sm text-subtle">No notifications yet</p>
                ) : (
                    <ul className="divide-y divide-line">
                        {items.map((n) => (
                            <li key={n.id} className="group relative px-3 py-2.5">
                                <span aria-hidden="true" className="mr-2 text-sm">
                                    {KIND_ICON[n.kind]}
                                </span>
                                <span className={`text-sm ${n.read ? 'text-subtle' : 'font-medium text-ink'}`}>
                                    {n.title}
                                </span>
                                {n.body && <p className="mt-0.5 pl-6 text-xs text-subtle">{n.body}</p>}
                                <button
                                    type="button"
                                    onClick={() => dispatch(dismissNotification(n.id))}
                                    aria-label={`Dismiss: ${n.title}`}
                                    className="absolute right-2 top-2 hidden rounded-md p-1 text-subtle transition hover:bg-raised hover:text-ink group-hover:block"
                                >
                                    <X className="h-3 w-3" aria-hidden="true" />
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    )
}
