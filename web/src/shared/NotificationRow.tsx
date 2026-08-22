import { Badge, Button, List, Typography } from 'antd'
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
        <List.Item
            actions={[
                <Button
                    key="dismiss"
                    type="text"
                    size="small"
                    aria-label={`Dismiss: ${title}`}
                    icon={<DeleteOutlined aria-hidden="true" />}
                    onClick={onDismiss}
                />
            ]}
        >
            <List.Item.Meta
                avatar={
                    <Badge dot={!read} offset={[-2, 4]}>
                        <NotificationIcon kind={kind} />
                    </Badge>
                }
                title={
                    <Typography.Text strong={!read} type={read ? 'secondary' : undefined}>
                        {title}
                    </Typography.Text>
                }
                description={body}
            />
        </List.Item>
    )
}
