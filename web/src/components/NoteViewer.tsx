import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { CopyButton } from './CopyButton'
import { Markdown } from './Markdown'
import { Tooltip } from './Tooltip'

export function NoteViewer({ path, onClose }: { path: string; onClose: () => void }) {
    const [content, setContent] = useState<string | null>(null)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        api.note(path)
            .then((n) => setContent(n.content))
            .catch((e: Error) => setError(e.message))
    }, [path])

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose()
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    }, [onClose])

    return (
        <aside className="flex min-h-0 min-w-0 flex-1 flex-col bg-surface">
            <header className="flex shrink-0 items-center justify-between gap-3 border-b border-line px-5 py-3">
                <span className="truncate font-mono text-xs text-subtle">{path}</span>
                <div className="flex shrink-0 items-center gap-2">
                    <Tooltip label="Copy raw">
                        <CopyButton
                            text={content ?? ''}
                            label="Copy raw"
                            toast="Note copied to clipboard"
                            className={`rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised ${content ? '' : 'pointer-events-none opacity-40'}`}
                        />
                    </Tooltip>
                    <button
                        onClick={onClose}
                        aria-label="Close note"
                        className="rounded-lg p-1.5 text-subtle transition hover:bg-raised hover:text-ink"
                    >
                        ✕
                    </button>
                </div>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
                {content && <Markdown>{content}</Markdown>}
            </div>
        </aside>
    )
}
