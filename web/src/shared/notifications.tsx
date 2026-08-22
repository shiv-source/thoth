import { Avatar } from 'antd'
import { BookOutlined, FileTextOutlined, MedicineBoxOutlined, MessageOutlined, SyncOutlined } from '@ant-design/icons'
import type { ComponentType } from 'react'
import type { NotificationKind } from '../store/slices/notificationsSlice'

// NOTIFICATION_ICONS is the single source for the per-kind icon used by both
// the header panel and the corner toasts.
export const NOTIFICATION_ICONS: Record<NotificationKind, ComponentType> = {
    sync: SyncOutlined,
    note: FileTextOutlined,
    rulebook: BookOutlined,
    chat: MessageOutlined,
    doctor: MedicineBoxOutlined
}

// NOTIFICATION_PALETTE pairs each kind with its semantic token pair (soft
// bg + strong glyph). The var() references resolve where the Avatar sits,
// inside the antd css-var scope.
const NOTIFICATION_PALETTE: Record<NotificationKind, { bg: string; color: string }> = {
    note: { bg: 'var(--ant-color-primary-bg)', color: 'var(--ant-color-primary)' },
    sync: { bg: 'var(--ant-color-success-bg)', color: 'var(--ant-color-success)' },
    rulebook: { bg: 'var(--ant-color-warning-bg)', color: 'var(--ant-color-warning)' },
    chat: { bg: 'var(--ant-color-info-bg)', color: 'var(--ant-color-info)' },
    doctor: { bg: 'var(--ant-color-error-bg)', color: 'var(--ant-color-error)' }
}

// NotificationIcon is a tinted circular avatar carrying the kind's icon.
export function NotificationIcon({ kind }: { kind: NotificationKind }) {
    const Icon = NOTIFICATION_ICONS[kind]
    const palette = NOTIFICATION_PALETTE[kind]
    return (
        <Avatar
            aria-hidden="true"
            size="small"
            icon={<Icon />}
            style={{ backgroundColor: palette.bg, color: palette.color }}
        />
    )
}
