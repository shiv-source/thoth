import { useEffect } from 'react'
import { Alert, Button, Skeleton } from 'antd'
import { CloseOutlined } from '@ant-design/icons'
import { fetchNote, selectNoteContent, selectNoteError, selectNoteLoading } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { CopyButton } from './CopyButton'
import { Markdown } from './Markdown'

// NoteViewer is the note reader, rendered inline in the Notes view's
// content area — the URL /notes/<path> owns the open note. Content lives
// in the note slice (fetched per path, stale responses discarded), so the
// note survives view switches and reopens instantly from the store.
export function NoteViewer({ path, onClose }: { path: string; onClose: () => void }) {
    const dispatch = useAppDispatch()
    const content = useAppSelector(selectNoteContent)
    const loading = useAppSelector(selectNoteLoading)
    const error = useAppSelector(selectNoteError)

    useEffect(() => {
        void dispatch(fetchNote(path))
    }, [path, dispatch])

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
                    <CopyButton
                        text={content ?? ''}
                        label="Copy raw"
                        toast="Note copied to clipboard"
                        className={`rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised ${content ? '' : 'pointer-events-none opacity-40'}`}
                    />
                    <Button
                        type="text"
                        size="small"
                        aria-label="Close note"
                        icon={<CloseOutlined aria-hidden="true" />}
                        onClick={onClose}
                    />
                </div>
            </header>
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
                {loading && <Skeleton active paragraph={{ rows: 6 }} />}
                {error && <Alert type="error" showIcon message={error} />}
                {content && <Markdown>{content}</Markdown>}
            </div>
        </aside>
    )
}
