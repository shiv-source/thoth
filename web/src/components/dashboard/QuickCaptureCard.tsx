import { useState } from 'react'
import { Button, Card, Input, Space } from 'antd'
import { InboxOutlined, PlusOutlined } from '@ant-design/icons'

// QuickCaptureCard is the Overview "Quick capture" widget: a single-line
// capture input that files the text into the inbox (mock until the capture
// endpoint lands). Enter or the button both capture.
export function QuickCaptureCard({ onCapture }: { onCapture: (text: string) => void }) {
    const [value, setValue] = useState('')

    const capture = () => {
        const text = value.trim()
        if (!text) return
        onCapture(text)
        setValue('')
    }

    return (
        <Card size="small" title="Quick capture">
            <Space.Compact block>
                <Input
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    onPressEnter={capture}
                    prefix={<InboxOutlined className="text-subtle" aria-hidden="true" />}
                    placeholder="Capture a note, link, or thought…"
                    aria-label="Quick capture"
                />
                <Button type="primary" onClick={capture} icon={<PlusOutlined aria-hidden="true" />}>
                    Capture
                </Button>
            </Space.Compact>
            <p className="mt-3 text-xs text-subtle">Enter to capture · lands in inbox/ — mock data (#17)</p>
        </Card>
    )
}
