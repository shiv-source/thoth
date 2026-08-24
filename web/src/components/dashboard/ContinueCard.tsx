import type { ComponentType } from 'react'
import { Avatar, Badge, Card, Flex, Listy, Space, Tag } from 'antd'
import {
    CalendarOutlined,
    InboxOutlined,
    LinkOutlined,
    MessageOutlined,
    ReadOutlined,
    RightOutlined
} from '@ant-design/icons'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'
import type { Conversation } from '../../api/client'
import { EmptyState } from '../../shared/EmptyState'
import { relativeDate } from '../../utils/time'

// RecentNote is one note in the resume strip — the wiki path, its display
// title, the note kind (drives the row icon), and when it was last touched.
export interface RecentNote {
    path: string
    title: string
    kind: string
    updatedAt: string
}

// The note-kind icon set matches the wiki's type: vocabulary so each resume
// row reads at a glance; unknown kinds fall back to the generic reader icon.
const kindIcon: Record<string, ComponentType<AntdIconProps>> = {
    capture: InboxOutlined,
    knowledge: ReadOutlined,
    link: LinkOutlined,
    meeting: CalendarOutlined
}

// The neutral kind label shown beside each resume row.
const kindLabel: Record<string, string> = {
    capture: 'Capture',
    knowledge: 'Knowledge',
    link: 'Link',
    meeting: 'Meeting'
}

// ContinueCard is the Overview "Continue where you left off" widget: recent
// chats and recently touched notes merged into one recency-sorted resume
// list, each row opening its chat or note. Rows are full-bleed so the hover
// surface reaches the card's padding edge.
export function ContinueCard({
    chats,
    notes,
    onOpenChat,
    onOpenNote
}: {
    chats: Conversation[]
    notes: RecentNote[]
    onOpenChat: (id: string) => void
    onOpenNote: (path: string) => void
}) {
    const rows = [
        ...notes.map((n) => ({
            key: n.path,
            title: n.title,
            subtitle: n.path,
            kind: kindLabel[n.kind] ?? 'Note',
            icon: kindIcon[n.kind] ?? ReadOutlined,
            updatedAt: n.updatedAt,
            onOpen: () => onOpenNote(n.path)
        })),
        ...chats.map((c) => ({
            key: c.id,
            title: c.title,
            subtitle: null as string | null,
            kind: 'Chat',
            icon: MessageOutlined,
            updatedAt: c.created_at,
            onOpen: () => onOpenChat(c.id)
        }))
    ]
        .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
        .slice(0, 6)

    return (
        <Card
            size="small"
            title="Continue where you left off"
            extra={<Badge count={rows.length} color="var(--ant-color-fill-secondary)" />}
            styles={{ body: { padding: '4px 16px 12px' } }}
        >
            {rows.length === 0 ? (
                <EmptyState
                    icon={<MessageOutlined className="h-5 w-5" aria-hidden="true" />}
                    title="No recent activity"
                    description="Your recent notes and chats will show up here."
                />
            ) : (
                <Listy
                    items={rows}
                    rowKey={(r) => r.key}
                    className="divide-y divide-line"
                    classNames={{ item: 'p-0!' }}
                    itemRender={(row) => (
                        <button
                            type="button"
                            onClick={row.onOpen}
                            className="group flex w-full items-center gap-4 rounded-lg -mx-4 px-4 py-3 text-left transition-colors hover:bg-raised"
                        >
                            <Avatar
                                shape="square"
                                size={36}
                                icon={<row.icon />}
                                style={{
                                    backgroundColor: 'var(--ant-color-fill-tertiary)',
                                    color: 'var(--ant-color-text-secondary)'
                                }}
                                className="shrink-0"
                            />
                            <Flex vertical gap={3} className="min-w-0 flex-1">
                                <span className="block truncate text-sm font-medium text-ink">{row.title}</span>
                                {row.subtitle && (
                                    <Space size={8} className="min-w-0">
                                        <Tag variant="filled" className="m-0! px-2! text-[11px] leading-5! text-faint">
                                            {row.kind}
                                        </Tag>
                                        <span className="truncate text-xs text-subtle">{row.subtitle}</span>
                                    </Space>
                                )}
                            </Flex>
                            <Space size={4} className="shrink-0">
                                <span className="text-xs text-faint">{relativeDate(row.updatedAt)}</span>
                                <RightOutlined
                                    className="h-3.5 w-3.5 text-subtle opacity-0 transition group-hover:opacity-100"
                                    aria-hidden="true"
                                />
                            </Space>
                        </button>
                    )}
                />
            )}
        </Card>
    )
}
