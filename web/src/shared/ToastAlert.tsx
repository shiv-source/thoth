import { useEffect } from 'react'
import { Alert, Button } from 'antd'
import { CloseOutlined } from '@ant-design/icons'
import type { NotificationKind } from '../store/slices/notificationsSlice'
import { NotificationIcon } from './notifications'

const TOAST_MS = 5000

// ToastAlert is one transient notification toast: an antd Alert that shows in
// the top-left corner and auto-dismisses (via onDismiss) after TOAST_MS.
export function ToastAlert({
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
            icon={<NotificationIcon kind={kind} />}
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
