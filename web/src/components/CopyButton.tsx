import { useEffect, useRef, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { useToast } from './Toast'

// CopyButton copies `text` to the clipboard and flips to a check for 2s,
// optionally surfacing a toast — the shared behavior behind the message,
// note, and code-block copy buttons. Placement and look come from the
// caller's className.
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
    const [copied, setCopied] = useState(false)
    const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
    const { toast: showToast } = useToast()

    useEffect(
        () => () => {
            if (timer.current) clearTimeout(timer.current)
        },
        []
    )

    const copy = async () => {
        try {
            await navigator.clipboard.writeText(text)
            setCopied(true)
            if (toast) showToast(toast, 'success')
            timer.current = setTimeout(() => setCopied(false), 2000)
        } catch {
            // Clipboard unavailable (permissions, non-secure context) — leave state untouched.
        }
    }

    return (
        <button type="button" onClick={() => void copy()} aria-label={copied ? 'Copied' : label} className={className}>
            {copied ? (
                <Check className="h-3.5 w-3.5 text-emerald-500" aria-hidden="true" />
            ) : (
                <Copy className="h-3.5 w-3.5" aria-hidden="true" />
            )}
        </button>
    )
}
