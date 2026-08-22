import { useRef } from 'react'
import { dismissNotification, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { ToastAlert } from './ToastAlert'

// NotificationToasts shows NEW notifications as transient alerts in the
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
            className="pointer-events-none absolute left-3 top-3 z-50 flex w-80 flex-col gap-2"
        >
            {toasts.map((n) => (
                <ToastAlert
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
