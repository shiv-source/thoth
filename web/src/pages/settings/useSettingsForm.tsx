import { useEffect, useRef, useState } from 'react'
import { App, Form } from 'antd'
import type { Settings } from '../../api/client'
import { fetchSettings, saveSettings, selectSettings } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { settingsBody } from './settingsBody'

// useSettingsForm is the shared settings sub-page machinery: one form seeded
// from the store's settings, a save handler that merges the page's fields
// into the full settings object (settingsBody) and reports via the transient
// status banner + a toast, and the fetch that loads settings on mount. Each
// sub-page owns its own instance so the fields it renders are the only ones
// it submits.
export function useSettingsForm() {
    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)
    const { message } = App.useApp()
    const [form] = Form.useForm<Settings>()
    // The save-feedback banner (Saved ✓ / error) is transient UI, not shared
    // state — it lives here; the store carries loading + data.
    const [status, setStatus] = useState<'idle' | 'saved' | 'error'>('idle')
    const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

    useEffect(() => {
        void dispatch(fetchSettings())
    }, [dispatch])

    // Seed the form when the store's settings arrive or are saved back;
    // setFieldsValue only touches the named fields, so mid-edit typing is
    // safe.
    useEffect(() => {
        if (settings.data) form.setFieldsValue(settings.data)
    }, [settings.data, form])

    useEffect(
        () => () => {
            if (savedTimer.current) clearTimeout(savedTimer.current)
        },
        []
    )

    const save = async (values: Settings) => {
        try {
            if (!settings.data) {
                setStatus('error')
                void message.error('Could not save settings')
                return
            }
            await dispatch(saveSettings(settingsBody(settings.data, values))).unwrap()
            // The echoed payload only carries the fields the user edited, so
            // re-sync from the server to restore full provider state.
            void dispatch(fetchSettings())
            setStatus('saved')
            void message.success('Settings saved')
            savedTimer.current = setTimeout(() => setStatus('idle'), 2000)
        } catch {
            setStatus('error')
            void message.error('Could not save settings')
        }
    }

    return {
        form,
        status,
        saving: settings.saving,
        hasError: settings.error !== null,
        save
    }
}
