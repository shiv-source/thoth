import type { ReactNode } from 'react'
import { LoadingOutlined } from '@ant-design/icons'

// StatusPill is a compact centered status chip for the chat header: an
// info tone (thinking/tool activity — spinning accent icon) or a warning
// tone (connection trouble — amber dot). The children carry the message,
// e.g. "Thinking…", "Reading <path>", "Connection lost — reconnecting…".
export function StatusPill({ tone = 'info', children }: { tone?: 'info' | 'warning'; children: ReactNode }) {
    return (
        <div className="flex justify-center py-1.5">
            <div
                className={`inline-flex max-w-full items-center gap-2 rounded-full border px-3 py-1 text-xs ${
                    tone === 'warning'
                        ? 'border-warning-border bg-warning-soft text-warning-ink'
                        : 'border-line bg-surface text-subtle'
                }`}
            >
                {tone === 'warning' ? (
                    <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-warning" />
                ) : (
                    <LoadingOutlined spin aria-hidden="true" className="text-accent" />
                )}
                <span className="min-w-0 truncate">{children}</span>
            </div>
        </div>
    )
}
