import { Alert, Button } from 'antd'

// SaveFooter is the settings save bar shared by every sub-page that saves
// through useSettingsForm: the transient saved/error feedback banner with the
// submit button beside it (one convention, one place — SettingsGeneralPage
// and the per-provider panels use the same block).
export function SaveFooter({
    status,
    saving,
    hasError,
    className = ''
}: {
    status: 'idle' | 'saved' | 'error'
    saving: boolean
    hasError: boolean
    className?: string
}) {
    return (
        <div className={`flex items-center justify-between gap-3 ${className}`}>
            <div className="min-w-0 pr-3">
                {status === 'saved' && <Alert type="success" showIcon title="Saved ✓" />}
                {(status === 'error' || hasError) && <Alert type="error" showIcon title="Could not save settings." />}
            </div>
            <Button type="primary" htmlType="submit" loading={saving}>
                Save
            </Button>
        </div>
    )
}
