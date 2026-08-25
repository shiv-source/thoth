import { useState } from 'react'
import { Button, Input } from 'antd'
import { SendOutlined } from '@ant-design/icons'
import { ConversationUsage } from './ConversationUsage'
import type { TokenUsage } from '../../ws/protocol'

export function Composer({
    onSend,
    onCancel,
    streaming,
    model,
    usage
}: {
    onSend: (text: string) => void
    onCancel: () => void
    streaming: boolean
    model?: string
    usage?: TokenUsage
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
            className="shrink-0 border-t border-line bg-app px-4 pb-3 pt-4"
        >
            <div className="flex items-end gap-2">
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
            </div>
            <div className="mt-2 flex items-center justify-between gap-3 px-1">
                <span className="text-xs text-faint">Enter to send · Shift+Enter for a new line</span>
                <span className="flex shrink-0 items-center gap-3">
                    {usage && <ConversationUsage usage={usage} />}
                    {model && (
                        <span className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-line bg-surface px-2 py-0.5 text-xs text-subtle">
                            <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-success" />
                            {model}
                        </span>
                    )}
                </span>
            </div>
        </form>
    )
}
