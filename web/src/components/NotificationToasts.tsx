import { useEffect, useRef } from 'react'
import { X } from 'lucide-react'
import { dismissNotification, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import type { NotificationKind } from '../store/slices/notificationsSlice'

const KIND_ICON: Record<NotificationKind, string> = {
    sync: '🔄',
    note: '📝',
    rulebook: '📜',
    chat: '💬',
    doctor: '🩺'
}

const TOAST_MS = 5000

// NotificationToasts shows NEW notifications as transient cards in the
// top-left corner. Notifications that existed on mount are considered
// history (they live in the bell panel) and are not re-toasted.
export function NotificationToasts() {
    const items = useAppSelector(selectNotifications)
    const dispatch = useAppDispatch()
    // ids that existed at mount (history — lives in the bell panel) are
    // never toasted; ids are added as they toast so nothing shows twice.
    const seen = useRef(new Set(items.map((n) => n.id)))

    const toasts = items.filter((n) => !seen.current.has(n.id))
    for (const t of toasts) seen.current.add(t.id)

    return (
        <div
            aria-label="Notifications"
            className="pointer-events-none fixed left-16 top-3 z-50 flex w-80 flex-col gap-2"
        >
            {toasts.map((n) => (
                <ToastCard
                    key={n.id}
                    kind={n.kind}
                    title={n.title}
                    body={n.body}
                    onDismiss={() => dispatch(dismissNotification(n.id))}
                />
            ))}
        </div>
    )
}

function ToastCard({
    kind,
    title,
    body,
    onDismiss
}: {
    kind: NotificationKind
    title: string
    body?: string
    onDismiss: () => void
}) {
    const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
    useEffect(() => {
        timer.current = setTimeout(onDismiss, TOAST_MS)
        return () => {
            if (timer.current) clearTimeout(timer.current)
        }
    }, [onDismiss])

    return (
        <div
            role="status"
            className="pointer-events-auto flex items-start gap-2 rounded-xl border border-line bg-surface px-3 py-2.5 shadow-lg"
        >
            <span aria-hidden="true" className="mt-0.5 text-sm">
                {KIND_ICON[kind]}
            </span>
            <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-ink">{title}</p>
                {body && <p className="mt-0.5 text-xs text-subtle">{body}</p>}
            </div>
            <button
                type="button"
                onClick={onDismiss}
                aria-label={`Dismiss: ${title}`}
                className="rounded-md p-1 text-subtle transition hover:bg-raised hover:text-ink"
            >
                <X className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
        </div>
    )
}
