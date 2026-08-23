import { useEffect } from 'react'
import { navigateSegment, useViewRoute } from '../../hooks/useView'
import { SettingsDoctorPage } from './SettingsDoctorPage'
import { SettingsGeneralPage } from './SettingsGeneralPage'
import { SettingsProvidersPage } from './SettingsProvidersPage'
import { SettingsSyncPage } from './SettingsSyncPage'
import type { SettingsSection } from './SettingsShell'

// The settings section routes, mirroring the rail in SettingsShell. The
// segment rides the URL (#/settings/<section>) so it survives reloads and
// back/forward.
const SECTIONS: SettingsSection[] = ['general', 'providers', 'sync', 'doctor']

function sectionFromSegment(segment: string | null): SettingsSection {
    return SECTIONS.includes(segment as SettingsSection) ? (segment as SettingsSection) : 'general'
}

// SettingsPage dispatches /settings/<section> to the matching sub-page. An
// unknown or missing segment falls back to General — and the default is
// written into the URL so the route is always explicit.
export function SettingsPage() {
    const { segment } = useViewRoute()
    const section = sectionFromSegment(segment)

    useEffect(() => {
        if (section !== segment) navigateSegment('settings', section)
    }, [section, segment])

    switch (section) {
        case 'providers':
            return <SettingsProvidersPage />
        case 'sync':
            return <SettingsSyncPage />
        case 'doctor':
            return <SettingsDoctorPage />
        default:
            return <SettingsGeneralPage />
    }
}
