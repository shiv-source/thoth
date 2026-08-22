import type { ReactNode } from 'react'
import { Menu } from 'antd'
import { ApiOutlined, BranchesOutlined, MedicineBoxOutlined, SettingOutlined } from '@ant-design/icons'
import { AppHeader } from '../../shared/AppHeader'
import { navigateSegment } from '../../hooks/useView'

export type SettingsSection = 'general' | 'providers' | 'git' | 'doctor'

// The settings sub-page rail: one entry per /settings/<section> route. Icons
// are decorative — the labels carry the menu's accessible names.
const ITEMS: { key: SettingsSection; label: string; icon: ReactNode }[] = [
    { key: 'general', label: 'General', icon: <SettingOutlined aria-hidden="true" /> },
    { key: 'providers', label: 'Providers', icon: <ApiOutlined aria-hidden="true" /> },
    { key: 'git', label: 'Git remote', icon: <BranchesOutlined aria-hidden="true" /> },
    { key: 'doctor', label: 'Doctor', icon: <MedicineBoxOutlined aria-hidden="true" /> }
]

// SettingsShell is the shared settings layout: the header, a one-line
// description, a left rail navigating between the four /settings/<section>
// routes, and the section page as content. The rail stays put while the
// content scrolls; every section shares the same centered column width so
// the pages read as one system.
export function SettingsShell({ active, children }: { active: SettingsSection; children: ReactNode }) {
    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <AppHeader title="Settings" />
            <div className="flex min-h-0 w-full flex-1 flex-col px-6 py-5">
                <p className="mb-5 max-w-2xl text-sm text-subtle">
                    Configure your knowledge base, model providers, and GitHub sync — or run the installation health
                    checks.
                </p>
                <div className="flex min-h-0 flex-1 gap-6">
                    <nav className="w-48 shrink-0" aria-label="Settings sections">
                        <Menu
                            mode="inline"
                            items={ITEMS}
                            selectedKeys={[active]}
                            onClick={({ key }) => navigateSegment('settings', key)}
                            className="settings-menu"
                            style={{ borderInlineEnd: 'none' }}
                        />
                    </nav>
                    <div className="mx-auto w-full max-w-4xl min-h-0 min-w-0 flex-1 overflow-y-auto px-1">
                        {children}
                    </div>
                </div>
            </div>
        </div>
    )
}
