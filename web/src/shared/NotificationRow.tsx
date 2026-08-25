import { Badge, Button, Typography } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import type { NotificationKind } from '../store/slices/notificationsSlice'
import { NotificationIcon } from './notifications'

// NotificationRow is one notification list item: a tinted icon (unread dot
// when not read), the title/body, and the dismiss action.
export function NotificationRow({
    title,
    body,
    read,
    kind,
    onDismiss
}: {
    title: string
    body?: string
    read: boolean
    kind: NotificationKind
    onDismiss: () => void
}) {
    return (
        <div className="flex items-start px-3 py-2">
            <Badge dot={!read} offset={[-2, 4]} className="mr-2! shrink-0">
                <NotificationIcon kind={kind} />
            </Badge>
            <div className="min-w-0 flex-1">
                <Typography.Text strong={!read} type={read ? 'secondary' : undefined} className="block truncate">
                    {title}
                </Typography.Text>
                {body && <div className="truncate text-xs text-subtle">{body}</div>}
            </div>
            <Button
                type="text"
                size="small"
                aria-label={`Dismiss: ${title}`}
                icon={<DeleteOutlined aria-hidden="true" />}
                onClick={onDismiss}
            />
        </div>
    )
}
