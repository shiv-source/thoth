import { Badge } from 'antd'
import { selectHealth, selectHealthLoading } from '../../store'
import { useAppSelector } from '../../store/hooks'

// HealthFooter is the bottom status bar: an antd status dot with a one-line
// reason, and the app version on the right.
export function HealthFooter() {
    const health = useAppSelector(selectHealth)
    const loading = useAppSelector(selectHealthLoading)

    let status: 'processing' | 'success' | 'error' = 'processing'
    let label = 'Checking…'
    if (!loading) {
        if (health && health.backend.api_key_configured && health.wiki.exists) {
            status = 'success'
            label = 'All systems go'
        } else if (health && !health.backend.api_key_configured) {
            status = 'error'
            label = 'API key not configured'
        } else if (health && !health.wiki.exists) {
            status = 'error'
            label = 'Wiki missing'
        } else {
            status = 'error'
            label = 'Server unreachable'
        }
    }

    return (
        <footer className="flex h-10 shrink-0 items-center justify-between border-t border-line px-3">
            <Badge status={status} text={<span className="truncate text-[11px] text-subtle">{label}</span>} />
            <span className="shrink-0 text-[11px] text-subtle">v{health?.version ?? '…'}</span>
        </footer>
    )
}
