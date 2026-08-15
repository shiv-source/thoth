import type { NotificationKind } from '../store/slices/notificationsSlice'

// NOTIFICATION_ICONS is the single source for the per-kind icon used by both
// the header panel and the corner toasts.
export const NOTIFICATION_ICONS: Record<NotificationKind, string> = {
    sync: '🔄',
    note: '📝',
    rulebook: '📜',
    chat: '💬',
    doctor: '🩺'
}

export function NotificationIcon({ kind }: { kind: NotificationKind }) {
    return (
        <span aria-hidden="true" className="text-sm">
            {NOTIFICATION_ICONS[kind]}
        </span>
    )
}
