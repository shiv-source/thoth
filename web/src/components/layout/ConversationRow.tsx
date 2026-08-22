import { Button, List, Tooltip } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import type { Conversation } from '../../api/client'

// ConversationRow is one history row: the title with its relative date, the
// active-conversation highlight, and a delete action that never bubbles into
// the row's open navigation.
export function ConversationRow({
    conversation,
    active,
    onOpen,
    onDelete
}: {
    conversation: Conversation
    active: boolean
    onOpen: () => void
    onDelete: () => void
}) {
    return (
        <List.Item
            aria-current={active ? 'true' : undefined}
            onClick={onOpen}
            className={`cursor-pointer rounded-md px-2 ${active ? 'bg-accent-soft' : 'hover:bg-raised'}`}
            actions={[
                <Tooltip key="delete" title="Delete chat">
                    <Button
                        type="text"
                        size="small"
                        danger
                        aria-label={`Delete ${conversation.title}`}
                        icon={<DeleteOutlined aria-hidden="true" />}
                        onClick={(e) => {
                            e.stopPropagation()
                            onDelete()
                        }}
                    />
                </Tooltip>
            ]}
        >
            <span className={`min-w-0 truncate text-sm ${active ? 'font-medium text-accent' : ''}`}>
                {conversation.title}
            </span>
            <span className="ml-2 shrink-0 text-[11px] text-subtle">{relativeDate(conversation.created_at)}</span>
        </List.Item>
    )
}

// relativeDate renders "3 days ago"-style labels with a plain-date fallback.
const rtf =
    typeof Intl !== 'undefined' && 'RelativeTimeFormat' in Intl
        ? new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
        : null

function relativeDate(iso: string): string {
    const then = new Date(iso).getTime()
    if (Number.isNaN(then)) return iso
    if (!rtf) return new Date(iso).toLocaleDateString()
    const seconds = Math.round((then - Date.now()) / 1000)
    const abs = Math.abs(seconds)
    if (abs < 60) return rtf.format(seconds, 'second')
    const minutes = Math.round(seconds / 60)
    if (Math.abs(minutes) < 60) return rtf.format(minutes, 'minute')
    const hours = Math.round(minutes / 60)
    if (Math.abs(hours) < 24) return rtf.format(hours, 'hour')
    const days = Math.round(hours / 24)
    if (Math.abs(days) < 30) return rtf.format(days, 'day')
    const months = Math.round(days / 30)
    if (Math.abs(months) < 12) return rtf.format(months, 'month')
    return rtf.format(Math.round(months / 12), 'year')
}
