import { Button, Listy } from 'antd'
import { BellOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons'
import { dismissNotification, markAllRead, selectNotifications } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { NotificationRow } from './NotificationRow'
import { EmptyState } from './EmptyState'

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
                <EmptyState
                    icon={<BellOutlined aria-hidden="true" />}
                    title="No notifications yet"
                    description="Wiki changes, sync results, and doctor warnings land here."
                    className="py-6"
                />
            ) : (
                <Listy
                    items={items}
                    rowKey="id"
                    className="max-h-80 divide-y divide-line overflow-y-auto"
                    classNames={{ item: 'p-0!' }}
                    itemRender={(n) => (
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
