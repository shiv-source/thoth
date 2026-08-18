import { useEffect, useRef, useState } from 'react'
import { App, Button, Layout, List, Skeleton, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import type { Conversation } from '../api/client'
import { navigate } from '../hooks/useConversationRoute'
import { deleteConversation, fetchConversations, selectConversations } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'

// ChatsList renders the conversation history grouped by day in one antd
// List per group. Clicking an item navigates to /chat/<id>; the route hook
// then loads and pins it. The list re-fetches on every URL change so a
// freshly created chat appears without extra plumbing (turn_done already
// updates the URL).
function ChatsList() {
    const dispatch = useAppDispatch()
    const { list: conversations, error } = useAppSelector(selectConversations)
    const [pathname, setPathname] = useState(window.location.pathname)
    const listRef = useRef<HTMLDivElement>(null)
    const { message } = App.useApp()

    useEffect(() => {
        const onPop = () => setPathname(window.location.pathname)
        window.addEventListener('popstate', onPop)
        return () => window.removeEventListener('popstate', onPop)
    }, [])

    useEffect(() => {
        void dispatch(fetchConversations())
    }, [pathname, dispatch])

    const activeConvID = /^\/chat\/([0-9a-fA-F-]{36})$/.exec(pathname)?.[1] ?? null

    // Keep the active item in view when the list re-renders.
    useEffect(() => {
        listRef.current?.querySelector('[aria-current="true"]')?.scrollIntoView({ block: 'nearest' })
    }, [conversations, activeConvID])

    if (error) {
        return (
            <div className="px-3 pb-2">
                <p className="text-sm text-red-600">Could not load conversations</p>
                <Button
                    type="link"
                    size="small"
                    className="h-auto p-0"
                    onClick={() => void dispatch(fetchConversations())}
                >
                    Retry
                </Button>
            </div>
        )
    }
    if (conversations === null) {
        return (
            <div className="px-3 pb-2">
                <Skeleton active title={false} paragraph={{ rows: 4 }} />
            </div>
        )
    }
    if (conversations.length === 0) {
        return <p className="px-3 pb-2 text-sm text-subtle">No conversations yet — your chats will appear here.</p>
    }

    const groups = groupByDay(conversations)
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
                    renderItem={(c) => {
                        const active = c.id === activeConvID
                        return (
                            <List.Item
                                aria-current={active ? 'true' : undefined}
                                onClick={() => navigate(`/chat/${c.id}`)}
                                className={`cursor-pointer rounded-md px-2 ${active ? 'bg-accent-soft' : 'hover:bg-raised'}`}
                                actions={[
                                    <Tooltip key="delete" title="Delete chat">
                                        <Button
                                            type="text"
                                            size="small"
                                            danger
                                            aria-label={`Delete ${c.title}`}
                                            icon={<DeleteOutlined aria-hidden="true" />}
                                            onClick={(e) => {
                                                e.stopPropagation()
                                                void (async () => {
                                                    try {
                                                        await dispatch(deleteConversation(c.id)).unwrap()
                                                        void message.success('Conversation deleted')
                                                        if (c.id === activeConvID) navigate('/chat')
                                                    } catch {
                                                        void message.error('Could not delete the conversation')
                                                    }
                                                })()
                                            }}
                                        />
                                    </Tooltip>
                                ]}
                            >
                                <span className={`min-w-0 truncate text-sm ${active ? 'font-medium text-accent' : ''}`}>
                                    {c.title}
                                </span>
                                <span className="ml-2 shrink-0 text-[11px] text-subtle">
                                    {relativeDate(c.created_at)}
                                </span>
                            </List.Item>
                        )
                    }}
                />
            ))}
        </div>
    )
}

// groupByDay buckets conversations into display groups, newest first (the
// API already orders by created_at DESC).
function groupByDay(conversations: Conversation[]): [string, Conversation[]][] {
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
    const todayItems: Conversation[] = []
    const yesterdayItems: Conversation[] = []
    const weekItems: Conversation[] = []
    const olderItems: Conversation[] = []
    for (const c of conversations) {
        const t = new Date(c.created_at).getTime()
        if (Number.isNaN(t)) {
            olderItems.push(c)
        } else if (t >= today) {
            todayItems.push(c)
        } else if (t >= today - 86400000) {
            yesterdayItems.push(c)
        } else if (t >= today - 7 * 86400000) {
            weekItems.push(c)
        } else {
            olderItems.push(c)
        }
    }
    const buckets: [string, Conversation[]][] = [
        ['Today', todayItems],
        ['Yesterday', yesterdayItems],
        ['Previous 7 days', weekItems],
        ['Older', olderItems]
    ]
    return buckets.filter(([, items]) => items.length > 0)
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

// Sidebar is the chat view's history column: chats label, New chat
// primary button, and the grouped conversation list. The brand wordmark
// and health footer live in AppSider.
export function Sidebar() {
    return (
        <Layout.Sider width={288} theme="light" className="border-r border-line">
            <div className="flex h-full flex-col">
                <div className="shrink-0 px-3 pb-1 pt-3">
                    <span className="text-[11px] font-semibold uppercase tracking-wider text-subtle">Chats</span>
                </div>
                <div className="shrink-0 px-3 pb-2">
                    <Button
                        type="primary"
                        block
                        icon={<PlusOutlined aria-hidden="true" />}
                        onClick={() => navigate('/chat')}
                    >
                        New chat
                    </Button>
                </div>
                <div className="min-h-0 flex-1">
                    <ChatsList />
                </div>
            </div>
        </Layout.Sider>
    )
}
