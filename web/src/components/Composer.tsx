import { useState } from 'react'
import { Button, Input } from 'antd'
import { SendOutlined } from '@ant-design/icons'

export function Composer({
    onSend,
    onCancel,
    streaming
}: {
    onSend: (text: string) => void
    onCancel: () => void
    streaming: boolean
}) {
    // The draft is deliberately local state, not Redux: dispatching on
    // every keystroke would re-render the store tree for no benefit.
    const [text, setText] = useState('')

    const submit = () => {
        const t = text.trim()
        if (!t) return
        // Sending while a turn streams is allowed: the server cancels the
        // in-flight turn and starts the new one (supersede).
        setText('')
        onSend(t)
    }

    return (
        <form
            onSubmit={(e) => {
                e.preventDefault()
                submit()
            }}
            className="flex shrink-0 items-end gap-2 border-t border-line bg-app px-4 py-4"
        >
            <Input.TextArea
                value={text}
                onChange={(e) => setText(e.target.value)}
                onPressEnter={(e) => {
                    if (!e.shiftKey) {
                        e.preventDefault()
                        submit()
                    }
                }}
                autoSize={{ minRows: 2, maxRows: 8 }}
                placeholder="Ask your wiki anything — or tell Thoth to save something…"
                className="flex-1"
            />
            {streaming ? (
                <Button size="large" onClick={onCancel}>
                    Stop
                </Button>
            ) : (
                <Button
                    type="primary"
                    size="large"
                    htmlType="submit"
                    disabled={!text.trim()}
                    icon={<SendOutlined aria-hidden="true" />}
                >
                    Send
                </Button>
            )}
        </form>
    )
}
