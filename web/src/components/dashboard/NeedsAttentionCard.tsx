import type { ComponentType } from 'react'
import { Avatar, Badge, Card, Flex, Listy } from 'antd'
import { CheckSquareOutlined, InboxOutlined, RightOutlined, SyncOutlined } from '@ant-design/icons'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'
import { EmptyState } from '../../shared/EmptyState'

// AttentionItem is one "Needs attention" row: the subject, a one-line
// detail, a tone (tints the icon tile), and the kind that picks its icon.
export interface AttentionItem {
    id: string
    title: string
    detail: string
    tone: 'default' | 'warning' | 'danger'
    kind: 'capture' | 'todo' | 'sync'
}

const kindIcon: Record<AttentionItem['kind'], ComponentType<AntdIconProps>> = {
    capture: InboxOutlined,
    todo: CheckSquareOutlined,
    sync: SyncOutlined
}

// Tone → antd css-var colors (the same palette the theme.tsx tokens bridge).
const toneStyle: Record<AttentionItem['tone'], { backgroundColor: string; color: string }> = {
    default: { backgroundColor: 'var(--ant-color-fill-tertiary)', color: 'var(--ant-color-text-secondary)' },
    warning: { backgroundColor: 'var(--ant-color-warning-bg)', color: 'var(--ant-color-warning-text)' },
    danger: { backgroundColor: 'var(--ant-color-error-bg)', color: 'var(--ant-color-error)' }
}

// NeedsAttentionCard is the Overview "Needs attention" widget: the items
// that want the user's input (waiting captures, open todos, sync debt), each
// opening the relevant view. Rows are full-bleed like the resume strip.
export function NeedsAttentionCard({ items, onOpen }: { items: AttentionItem[]; onOpen: () => void }) {
    return (
        <Card
            size="small"
            title="Needs attention"
            styles={{ body: { padding: '4px 16px 12px' } }}
            extra={
                <Badge
                    count={items.length}
                    color={items.length > 0 ? 'var(--ant-color-warning)' : 'var(--ant-color-fill-secondary)'}
                />
            }
        >
            {items.length === 0 ? (
                <EmptyState
                    icon={<CheckSquareOutlined className="h-5 w-5" aria-hidden="true" />}
                    title="All caught up"
                    description="Nothing needs your attention right now."
                />
            ) : (
                <Listy
                    items={items}
                    rowKey={(i) => i.id}
                    className="divide-y divide-line"
                    classNames={{ item: 'p-0!' }}
                    itemRender={(item) => {
                        const Icon = kindIcon[item.kind]
                        return (
                            <button
                                type="button"
                                onClick={onOpen}
                                className="group flex w-full items-center gap-4 rounded-lg -mx-4 px-4 py-3 text-left transition-colors hover:bg-raised"
                            >
                                <Avatar
                                    shape="square"
                                    size={36}
                                    icon={<Icon />}
                                    style={toneStyle[item.tone]}
                                    className="shrink-0"
                                />
                                <Flex vertical gap={3} className="min-w-0 flex-1">
                                    <span className="block truncate text-sm font-medium text-ink">{item.title}</span>
                                    <span className="block truncate text-xs text-subtle">{item.detail}</span>
                                </Flex>
                                <RightOutlined
                                    className="h-3.5 w-3.5 shrink-0 text-subtle opacity-0 transition group-hover:opacity-100"
                                    aria-hidden="true"
                                />
                            </button>
                        )
                    }}
                />
            )}
        </Card>
    )
}
