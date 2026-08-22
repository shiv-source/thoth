import { Button, Empty, List } from 'antd'
import { CheckOutlined, CloseOutlined } from '@ant-design/icons'
import { dismissNotification, markAllRead, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { NotificationRow } from './NotificationRow'

// NotificationPanel is the Popover content for the header bell: a header
// row with mark-all-read/close, then the notification list. Dismissing an
// item removes it from the store ring immediately.
export function NotificationPanel({ onClose }: { onClose: () => void }) {
    const items = useAppSelector(selectNotifications)
    const dispatch = useAppDispatch()

    return (
        <div role="dialog" aria-label="Notifications" className="w-80">
            <div className="flex items-center justify-between border-b border-line px-3 py-1.5">
                <span className="text-sm font-medium text-ink">Notifications</span>
                <div className="flex items-center">
                    <Button
                        type="text"
                        size="small"
                        aria-label="Mark all as read"
                        icon={<CheckOutlined aria-hidden="true" />}
                        onClick={() => dispatch(markAllRead())}
                    />
                    <Button
                        type="text"
                        size="small"
                        aria-label="Close notifications"
                        icon={<CloseOutlined aria-hidden="true" />}
                        onClick={onClose}
                    />
                </div>
            </div>
            {items.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No notifications yet" className="py-6" />
            ) : (
                <List
                    size="small"
                    className="max-h-80 overflow-y-auto"
                    dataSource={items}
                    renderItem={(n) => (
                        <NotificationRow
                            title={n.title}
                            body={n.body}
                            read={n.read}
                            kind={n.kind}
                            onDismiss={() => dispatch(dismissNotification(n.id))}
                        />
                    )}
                />
            )}
        </div>
    )
}
