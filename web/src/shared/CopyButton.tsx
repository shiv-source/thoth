import { useEffect, useRef, useState } from 'react'
import { App, Button } from 'antd'
import { CheckOutlined, CopyOutlined } from '@ant-design/icons'

// CopyButton is the shared copy control: an antd text button that writes
// to the clipboard, flips to a check for two seconds, and optionally
// surfaces a message toast. Layout comes from the caller's className.
export function CopyButton({
    text,
    label,
    toast,
    className
}: {
    text: string
    label: string
    toast?: string
    className?: string
}) {
    const { message } = App.useApp()
    const [copied, setCopied] = useState(false)
    const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

    useEffect(
        () => () => {
            if (timer.current) clearTimeout(timer.current)
        },
        []
    )

    const copy = async () => {
        try {
            await navigator.clipboard.writeText(text)
        } catch {
            return
        }
        setCopied(true)
        if (toast) void message.success(toast)
        timer.current = setTimeout(() => setCopied(false), 2000)
    }

    return (
        <Button
            type="text"
            size="small"
            aria-label={copied ? 'Copied' : label}
            icon={
                copied ? (
                    <CheckOutlined aria-hidden="true" className="text-emerald-500" />
                ) : (
                    <CopyOutlined aria-hidden="true" />
                )
            }
            onClick={() => void copy()}
            className={className}
        />
    )
}
