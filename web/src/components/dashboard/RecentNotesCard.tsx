import { Badge, Card, Tag } from 'antd'
import { FileTextOutlined } from '@ant-design/icons'
import type { RecentNote } from './ContinueCard'
import { relativeDate } from '../../utils/time'

// The note-kind label set matches ContinueCard's vocabulary so the two
// widgets read identically; unknown kinds fall back to "Note".
const kindLabel: Record<string, string> = {
    capture: 'Capture',
    knowledge: 'Knowledge',
    link: 'Link',
    meeting: 'Meeting'
}

// RecentNotesCard is the Overview "Recent notes" grid: the latest wiki notes
// as bordered tiles (title, kind, path, recency) that open the note. It sits
// beside the Continue strip to show the full recent surface, not just the
// resume-worthy slice.
export function RecentNotesCard({ notes, onOpen }: { notes: RecentNote[]; onOpen: (path: string) => void }) {
    return (
        <Card
            size="small"
            title="Recent notes"
            extra={<Badge count={notes.length} color="var(--ant-color-fill-secondary)" />}
        >
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {notes.map((n) => (
                    <button
                        key={n.path}
                        type="button"
                        onClick={() => onOpen(n.path)}
                        className="group flex flex-col gap-2 rounded-lg border border-line bg-raised p-3 text-left transition-colors hover:border-accent-border"
                    >
                        <span className="flex items-center gap-2">
                            <FileTextOutlined className="shrink-0 text-sm text-faint" aria-hidden="true" />
                            <span className="truncate text-sm font-medium text-ink">{n.title}</span>
                        </span>
                        <span className="truncate font-mono text-xs text-subtle">{n.path}</span>
                        <span className="flex items-center justify-between gap-2">
                            <Tag variant="filled" className="m-0! px-2! text-[11px] leading-5! text-faint">
                                {kindLabel[n.kind] ?? 'Note'}
                            </Tag>
                            <span className="shrink-0 text-xs text-faint">{relativeDate(n.updatedAt)}</span>
                        </span>
                    </button>
                ))}
            </div>
            <p className="mt-3 text-xs text-subtle">mock data — index updated_at endpoint</p>
        </Card>
    )
}
