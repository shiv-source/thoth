import { Alert, Empty, Skeleton } from 'antd'
import { Markdown } from '../../shared/Markdown'
import { noteUrl } from './notePaths'

// NoteBody renders the viewer's content area: markdown notes render through
// Markdown (loading/error states included), image attachments inline, and any
// other file type gets the "can't be previewed" state.
export function NoteBody({
    isNote,
    isImage,
    loading,
    error,
    content,
    path
}: {
    isNote: boolean
    isImage: boolean
    loading: boolean
    error: string | null
    content: string | null
    path: string
}) {
    if (isNote) {
        return (
            <>
                {loading && <Skeleton active paragraph={{ rows: 6 }} />}
                {error && <Alert type="error" showIcon title={error} />}
                {content && <Markdown>{content}</Markdown>}
            </>
        )
    }
    if (isImage) {
        return <img src={noteUrl(path)} alt={path} className="max-w-full rounded-lg border border-line" />
    }
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="This file type can't be previewed." />
}
