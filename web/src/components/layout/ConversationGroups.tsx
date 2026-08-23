import type { RefObject } from 'react'
import { Listy } from 'antd'
import type { Conversation } from '../../api/client'
import { ConversationRow } from './ConversationRow'

// ConversationGroups renders the day-grouped conversation history, one group
// label followed by its rows. Pure presentation — the grouped data and the
// open/delete events flow in from ChatsList.
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
                <div key={label}>
                    <div className="px-1 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-subtle">
                        {label}
                    </div>
                    <Listy
                        items={items}
                        rowKey="id"
                        classNames={{ item: 'p-0!' }}
                        itemRender={(c) => (
                            <ConversationRow
                                conversation={c}
                                active={c.id === activeConvID}
                                onOpen={() => onOpen(c.id)}
                                onDelete={() => onDelete(c)}
                            />
                        )}
                    />
                </div>
            ))}
        </div>
    )
}
