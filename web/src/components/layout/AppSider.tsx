import { Layout, Menu } from 'antd'
import {
    FileTextOutlined,
    DashboardOutlined,
    MessageOutlined,
    SearchOutlined,
    SettingOutlined
} from '@ant-design/icons'
import { navigateView, useView, type View } from '../../hooks/useView'
import { Logo } from '../../shared/Logo'
import { HealthFooter } from './HealthFooter'

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
        <Layout.Sider width={232} theme="light" className="bg-surface">
            <div className="flex h-full flex-col">
                <div className="flex h-14 shrink-0 items-center px-4">
                    <Logo />
                </div>
                <Menu
                    mode="inline"
                    items={ITEMS}
                    selectedKeys={[view]}
                    onClick={({ key }) => navigateView(key as View)}
                    className="min-h-0 flex-1 overflow-y-auto px-2 py-1"
                    style={{ borderInlineEnd: 'none' }}
                />
                <HealthFooter />
            </div>
        </Layout.Sider>
    )
}
