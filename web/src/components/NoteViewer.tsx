import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { api } from '../api/client'
import { useToast } from './Toast'

export function NoteViewer({ path, onClose, inline = false }: { path: string; onClose: () => void; inline?: boolean }) {
    const [content, setContent] = useState<string | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [copied, setCopied] = useState(false)
    const copyTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
    const { toast } = useToast()

    useEffect(() => {
        api.note(path)
            .then((n) => setContent(n.content))
            .catch((e: Error) => setError(e.message))
        setCopied(false)
        return () => {
            if (copyTimer.current) clearTimeout(copyTimer.current)
        }
    }, [path])

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose()
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    }, [onClose])

    const copy = async () => {
        if (!content) return
        try {
            await navigator.clipboard.writeText(content)
            setCopied(true)
            toast('Note copied to clipboard', 'success')
            copyTimer.current = setTimeout(() => setCopied(false), 2000)
        } catch {
            // Clipboard unavailable (permissions, non-secure context) — leave state untouched.
        }
    }

    return (
        <aside
            className={
                inline
                    ? 'flex min-h-0 min-w-0 flex-1 flex-col bg-surface'
                    : 'fixed inset-y-0 right-0 z-40 flex w-[42rem] max-w-full animate-[slide-in-right_200ms_ease-out] flex-col border-l border-line bg-surface shadow-lg'
            }
        >
            <header className="flex shrink-0 items-center justify-between gap-3 border-b border-line px-5 py-3">
                <span className="truncate font-mono text-xs text-subtle">{path}</span>
                <button
                    onClick={onClose}
                    aria-label="Close note"
                    className="rounded-lg p-1.5 text-subtle transition hover:bg-raised hover:text-ink"
                >
                    ✕
                </button>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
                {content && (
                    <div className="prose prose-sm max-w-none prose-headings:font-display prose-headings:text-heading prose-code:before:content-none prose-code:after:content-none prose-pre:rounded-lg dark:prose-invert">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
                    </div>
                )}
            </div>
            <footer className="flex shrink-0 items-center justify-end border-t border-line px-5 py-3">
                <button
                    onClick={() => void copy()}
                    disabled={!content}
                    className="rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised disabled:cursor-not-allowed disabled:opacity-40"
                >
                    {copied ? 'Copied' : 'Copy raw'}
                </button>
            </footer>
        </aside>
    )
}
