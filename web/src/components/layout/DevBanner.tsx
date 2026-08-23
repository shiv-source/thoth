import { Alert } from 'antd'

// Full-width warning strip rendered above everything while serve --dev is
// running, so a dev session can never be mistaken for the production one.
// Alert's banner mode defaults to type warning with the icon shown; the
// section flex:none stops the content from stretching, so the root
// justifyContent centers the icon + message as one group across the strip.
export function DevBanner({ dev, commit }: { dev: boolean; commit: string }) {
    if (!dev) return null
    const message = commit
        ? `Dev mode — data is stored in ~/.thoth/dev · ${commit}`
        : 'Dev mode — data is stored in ~/.thoth/dev'
    return (
        <Alert
            banner
            showIcon
            title={message}
            styles={{
                root: { display: 'flex', alignItems: 'center', justifyContent: 'center' },
                section: { flex: 'none' }
            }}
        />
    )
}
