import type { RefObject } from 'react'
import { List } from 'antd'
import type { Conversation } from '../../api/client'
import { ConversationRow } from './ConversationRow'

// ConversationGroups renders the day-grouped conversation history as one
// antd List per group, the group label as its header. Pure presentation —
// the grouped data and the open/delete events flow in from ChatsList.
export function ConversationGroups({
    groups,
    activeConvID,
    listRef,
    onOpen,
    onDelete
}: {
    groups: [string, Conversation[]][]
    activeConvID: string | null
    listRef: RefObject<HTMLDivElement | null>
    onOpen: (id: string) => void
    onDelete: (conversation: Conversation) => void
}) {
    return (
        <div ref={listRef} className="h-full overflow-y-auto px-2 pb-2">
            {groups.map(([label, items]) => (
                <List
                    key={label}
                    size="small"
                    header={
                        <span className="text-[11px] font-semibold uppercase tracking-wider text-subtle">{label}</span>
                    }
                    dataSource={items}
                    renderItem={(c) => (
                        <ConversationRow
                            conversation={c}
                            active={c.id === activeConvID}
                            onOpen={() => onOpen(c.id)}
                            onDelete={() => onDelete(c)}
                        />
                    )}
                />
            ))}
        </div>
    )
}
