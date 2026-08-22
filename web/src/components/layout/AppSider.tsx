import { Badge, Layout, Menu } from 'antd'
import {
    FileTextOutlined,
    DashboardOutlined,
    MessageOutlined,
    SearchOutlined,
    SettingOutlined
} from '@ant-design/icons'
import { navigateView, useView, type View } from '../../hooks/useView'
import { selectHealth, selectHealthLoading } from '../../store'
import { useAppSelector } from '../../store/hooks'

// Icons are decorative — the labels carry the menu's accessible names, so
// the icons are hidden from the accessibility tree.
const ITEMS = [
    { key: 'dashboard', icon: <DashboardOutlined aria-hidden="true" />, label: 'Dashboard' },
    { key: 'chat', icon: <MessageOutlined aria-hidden="true" />, label: 'Chat' },
    { key: 'notes', icon: <FileTextOutlined aria-hidden="true" />, label: 'Notes' },
    { key: 'search', icon: <SearchOutlined aria-hidden="true" />, label: 'Search' },
    { type: 'divider' as const },
    { key: 'settings', icon: <SettingOutlined aria-hidden="true" />, label: 'Settings' }
]

// AppSider is the app's persistent navigation: brand wordmark, the view
// menu, and the health footer. Views route through the URL, so back/forward
// and deep links keep working.
export function AppSider() {
    const view = useView()

    return (
        <Layout.Sider width={208} theme="light" className="border-r border-line">
            <div className="flex h-full flex-col">
                <div className="flex h-14 shrink-0 items-center gap-2.5 px-4">
                    <span className="text-xl leading-none">🦉</span>
                    <span className="font-display text-lg font-semibold tracking-tight text-heading">Thoth</span>
                </div>
                <Menu
                    mode="inline"
                    items={ITEMS}
                    selectedKeys={[view]}
                    onClick={({ key }) => navigateView(key as View)}
                    className="min-h-0 flex-1 overflow-y-auto px-2"
                    style={{ borderInlineEnd: 'none' }}
                />
                <HealthFooter />
            </div>
        </Layout.Sider>
    )
}

// HealthFooter is the bottom status bar: an antd status dot with a one-line
// reason, and the app version on the right.
function HealthFooter() {
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
