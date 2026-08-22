import { useEffect } from 'react'
import { Alert, Button, Empty, Skeleton } from 'antd'
import { CloseOutlined, DownloadOutlined } from '@ant-design/icons'
import { fetchNote, selectNoteContent, selectNoteError, selectNoteLoading } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CopyButton } from '../../shared/CopyButton'
import { Markdown } from '../../shared/Markdown'

// isNotePath reports whether a wiki path is a previewable Markdown note
// (.md or .markdown, case-insensitive — matching wiki.IsMarkdownPath). The
// tree only lists markdown, but attachments (images, scripts, …) are indexed
// by filename and reachable by search or direct URL; those render as an image
// preview or a download instead of raw bytes as Markdown.
function isNotePath(path: string): boolean {
    return /\.(?:md|markdown)$/i.test(path)
}

// isImagePath reports whether a wiki path is a previewable image attachment
// (.png/.jpg/.jpeg/.gif/.svg/.webp, case-insensitive — matching
// wiki.IsImagePath). Images render inline; every other attachment gets a
// download action.
function isImagePath(path: string): boolean {
    return /\.(?:png|jpe?g|gif|svg|webp)$/i.test(path)
}

// noteUrl is the raw-bytes URL for an attachment: the server wraps markdown
// in JSON but serves any other path as raw bytes (images inline, everything
// else as a download), so an <img> or download link can point straight at it.
function noteUrl(path: string): string {
    return `/api/notes?path=${encodeURIComponent(path)}`
}

// NoteViewer is the note reader, rendered inline in the Notes view's
// content area — the URL /notes/<path> owns the open note. Content lives
// in the note slice (fetched per path, stale responses discarded), so the
// note survives view switches and reopens instantly from the store.
export function NoteViewer({ path, onClose }: { path: string; onClose: () => void }) {
    const dispatch = useAppDispatch()
    const content = useAppSelector(selectNoteContent)
    const loading = useAppSelector(selectNoteLoading)
    const error = useAppSelector(selectNoteError)
    const isNote = isNotePath(path)
    const isImage = isImagePath(path)

    useEffect(() => {
        if (!isNote) return
        void dispatch(fetchNote(path))
    }, [path, isNote, dispatch])

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
                    {isNote ? (
                        <CopyButton
                            text={content ?? ''}
                            label="Copy raw"
                            toast="Note copied to clipboard"
                            className={`rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised ${content ? '' : 'pointer-events-none opacity-40'}`}
                        />
                    ) : (
                        <Button
                            type="text"
                            size="small"
                            href={noteUrl(path)}
                            download={path.split('/').pop()}
                            icon={<DownloadOutlined aria-hidden="true" />}
                        >
                            Download
                        </Button>
                    )}
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
                {isNote ? (
                    <>
                        {loading && <Skeleton active paragraph={{ rows: 6 }} />}
                        {error && <Alert type="error" showIcon message={error} />}
                        {content && <Markdown>{content}</Markdown>}
                    </>
                ) : isImage ? (
                    <img src={noteUrl(path)} alt={path} className="max-w-full rounded-lg border border-line" />
                ) : (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="This file type can't be previewed." />
                )}
            </div>
        </aside>
    )
}
