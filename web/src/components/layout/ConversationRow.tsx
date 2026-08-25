import { Button, Tooltip } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import type { Conversation } from '../../api/client'
import { relativeDate } from '../../utils/time'

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
        <div
            aria-current={active ? 'true' : undefined}
            onClick={onOpen}
            className={`flex cursor-pointer items-center rounded-md px-2 py-1.5 ${active ? 'bg-accent-soft' : 'hover:bg-raised'}`}
        >
            <span className={`min-w-0 flex-1 truncate text-sm ${active ? 'font-medium text-accent' : ''}`}>
                {conversation.title}
            </span>
            <span className="ml-2 shrink-0 text-[11px] text-subtle">{relativeDate(conversation.created_at)}</span>
            <span className="ml-2 shrink-0">
                <Tooltip title="Delete chat">
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
            </span>
        </div>
    )
}
