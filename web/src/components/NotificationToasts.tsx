import { useEffect, useRef } from 'react'
import { Alert, Button } from 'antd'
import { CloseOutlined } from '@ant-design/icons'
import { dismissNotification, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import type { NotificationKind } from '../store/slices/notificationsSlice'
import { NotificationIcon } from './notifications'

const TOAST_MS = 5000

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

function ToastAlert({
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
    useEffect(() => {
        const timer = setTimeout(onDismiss, TOAST_MS)
        return () => clearTimeout(timer)
    }, [onDismiss])

    return (
        <Alert
            className="pointer-events-auto shadow-lg"
            type="info"
            showIcon
            icon={<span aria-hidden="true">{<NotificationIcon kind={kind} />}</span>}
            message={title}
            description={body}
            closable
            closeIcon={
                <Button
                    type="text"
                    size="small"
                    aria-label={`Dismiss: ${title}`}
                    icon={<CloseOutlined aria-hidden="true" />}
                />
            }
            onClose={onDismiss}
        />
    )
}
