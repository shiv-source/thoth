import { useEffect, useRef, useState } from 'react'
import { App, Button, Skeleton } from 'antd'
import type { Conversation } from '../../api/client'
import { navigate } from '../../hooks/useConversationRoute'
import { deleteConversation, fetchConversations, selectConversations } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { ConversationGroups } from './ConversationGroups'

// ChatsList renders the conversation history grouped by day in one antd
// List per group (rows are ConversationRow). Clicking an item navigates to
// /chat/<id>; the route hook then loads and pins it. The list re-fetches on
// every URL change so a freshly created chat appears without extra plumbing
// (turn_done already updates the URL).
export function ChatsList() {
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

    const remove = async (c: Conversation) => {
        try {
            await dispatch(deleteConversation(c.id)).unwrap()
            void message.success('Conversation deleted')
            if (c.id === activeConvID) navigate('/chat')
        } catch {
            void message.error('Could not delete the conversation')
        }
    }

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
        <ConversationGroups
            groups={groups}
            activeConvID={activeConvID}
            listRef={listRef}
            onOpen={(id) => navigate(`/chat/${id}`)}
            onDelete={(c) => void remove(c)}
        />
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
