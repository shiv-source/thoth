import { Card, Listy } from 'antd'
import type { Conversation } from '../../api/client'

// RecentChatsCard is the Overview "Recent chats" widget: the three most
// recent conversations, each opening its /chat/<id> route on click.
export function RecentChatsCard({ chats, onOpen }: { chats: Conversation[]; onOpen: (id: string) => void }) {
    return (
        <Card size="small" title="Recent chats">
            {chats.length === 0 ? (
                <p className="text-sm text-subtle">No conversations yet</p>
            ) : (
                <Listy
                    items={chats}
                    rowKey="id"
                    className="divide-y divide-line"
                    classNames={{ item: 'p-0!' }}
                    itemRender={(c) => (
                        <div
                            onClick={() => onOpen(c.id)}
                            className="cursor-pointer truncate py-1 text-sm text-ink hover:bg-raised"
                        >
                            {c.title}
                        </div>
                    )}
                />
            )}
        </Card>
    )
}
