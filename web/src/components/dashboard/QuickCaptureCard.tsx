import { useState } from 'react'
import { App, Button, Card, Input, Space } from 'antd'
import { InboxOutlined, PlusOutlined } from '@ant-design/icons'
import { api } from '../../api/client'

// QuickCaptureCard is the Overview "Quick capture" widget: a single-line
// capture input that files the text into the inbox through the unified
// capture endpoint (the same save path the browser extension uses). Enter or
// the button both capture; the saved path is toasted and handed to the page
// so it can open the note.
export function QuickCaptureCard({ onCaptured }: { onCaptured?: (path: string) => void }) {
    const { message } = App.useApp()
    const [value, setValue] = useState('')
    const [saving, setSaving] = useState(false)

    const capture = async () => {
        const text = value.trim()
        if (!text) return
        setSaving(true)
        try {
            const res = await api.capture({ kind: 'note', text, folder: 'inbox' })
            message.success(`Captured to ${res.path}`)
            onCaptured?.(res.path)
            setValue('')
        } catch {
            message.error('Could not capture — is the server running?')
        } finally {
            setSaving(false)
        }
    }

    return (
        <Card size="small" title="Quick capture">
            <Space.Compact block>
                <Input
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    onPressEnter={() => void capture()}
                    prefix={<InboxOutlined className="text-subtle" aria-hidden="true" />}
                    placeholder="Capture a note, link, or thought…"
                    aria-label="Quick capture"
                />
                <Button
                    type="primary"
                    onClick={() => void capture()}
                    loading={saving}
                    icon={<PlusOutlined aria-hidden="true" />}
                >
                    Capture
                </Button>
            </Space.Compact>
            <p className="mt-3 text-xs text-subtle">Enter to capture · lands in inbox/</p>
        </Card>
    )
}
