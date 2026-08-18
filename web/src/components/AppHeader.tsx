import { useEffect } from 'react'
import { Badge, Button, Popover, Tooltip } from 'antd'
import { BellOutlined, SettingOutlined } from '@ant-design/icons'
import { NotificationPanel } from './NotificationPanel'
import { selectNotificationsOpen, selectUnreadCount, setNotificationsOpen } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'

// AppHeader is the per-view header: title on the left, the notification
// bell (antd Badge + Popover) and an optional settings shortcut on the
// right. Popover visibility lives in the ui slice so any view state
// survives view switches; Escape closes it like a modal.
export function AppHeader({ title, onOpenSettings }: { title: string; onOpenSettings?: () => void }) {
    const dispatch = useAppDispatch()
    const unread = useAppSelector(selectUnreadCount)
    const open = useAppSelector(selectNotificationsOpen)

    useEffect(() => {
        if (!open) return
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') dispatch(setNotificationsOpen(false))
        }
        document.addEventListener('keydown', onKey)
        return () => document.removeEventListener('keydown', onKey)
    }, [open, dispatch])

    return (
        <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-line bg-surface px-4">
            <h1 className="truncate text-sm font-medium text-ink">{title}</h1>
            <div className="flex shrink-0 items-center gap-1">
                <Popover
                    trigger="click"
                    placement="bottomRight"
                    open={open}
                    destroyOnHidden
                    onOpenChange={(next) => dispatch(setNotificationsOpen(next))}
                    content={<NotificationPanel onClose={() => dispatch(setNotificationsOpen(false))} />}
                >
                    <Badge count={unread} size="small" title={`${unread} unread`}>
                        <Button type="text" aria-label="Notifications" icon={<BellOutlined aria-hidden="true" />} />
                    </Badge>
                </Popover>
                {onOpenSettings && (
                    <Tooltip title="Settings">
                        <Button
                            type="text"
                            aria-label="Settings"
                            icon={<SettingOutlined aria-hidden="true" />}
                            onClick={onOpenSettings}
                        />
                    </Tooltip>
                )}
            </div>
        </header>
    )
}
