import { useEffect, useRef, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { api, type Conversation, type Health } from '../api/client'
import { navigate } from '../hooks/useConversationRoute'
import { useToast } from './Toast'
import { SearchPanel } from './SearchPanel'
import { WikiTree } from './WikiTree'

// ChatsList renders the conversation history grouped by day. Clicking an
// item navigates to /chat/<id>; the route hook then loads and pins it. The
// list re-fetches on every URL change so a freshly created chat appears
// without extra plumbing (turn_done already updates the URL).
function ChatsList() {
  const [conversations, setConversations] = useState<Conversation[] | null>(null)
  const [error, setError] = useState(false)
  const [pathname, setPathname] = useState(window.location.pathname)
  const listRef = useRef<HTMLUListElement>(null)
  const { toast } = useToast()

  useEffect(() => {
    const onPop = () => setPathname(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  useEffect(() => {
    api.listConversations()
      .then((r) => setConversations(r.conversations))
      .catch(() => setError(true))
  }, [pathname])

  const activeConvID = /^\/chat\/([0-9a-fA-F-]{36})$/.exec(pathname)?.[1] ?? null

  // Keep the active item in view when the list re-renders.
  useEffect(() => {
    listRef.current?.querySelector('[aria-current="true"]')?.scrollIntoView({ block: 'nearest' })
  }, [conversations, activeConvID])

  if (error) {
    return (
      <div className="px-3 pb-2">
        <p className="text-sm text-red-600 dark:text-red-400">Could not load conversations</p>
        <button onClick={() => setError(false)} className="mt-1 text-xs text-subtle underline">Retry</button>
      </div>
    )
  }
  if (conversations === null) {
    return (
      <div className="space-y-1.5 px-3 pb-2" aria-hidden="true">
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-7 animate-pulse rounded-md bg-raised" />
        ))}
      </div>
    )
  }
  if (conversations.length === 0) {
    return <p className="px-3 pb-2 text-sm text-subtle">No conversations yet — your chats will appear here.</p>
  }

  const groups = groupByDay(conversations)
  return (
    <ul ref={listRef} className="h-full space-y-0.5 overflow-y-auto px-1.5 pb-2">
      {groups.map(([label, items]) => (
        <li key={label}>
          <div className="px-2 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-subtle">
            {label}
          </div>
          <ul className="space-y-0.5">
            {items.map((c) => {
              const active = c.id === activeConvID
              return (
                <li key={c.id} className="group flex items-center">
                  <button
                    aria-current={active ? 'true' : undefined}
                    onClick={() => navigate(`/chat/${c.id}`)}
                    className={`flex min-w-0 flex-1 items-center gap-1.5 rounded-md py-1.5 pl-2 pr-1 text-left text-sm transition ${
                      active ? 'bg-accent-soft font-medium text-accent' : 'text-ink hover:bg-raised'
                    }`}
                  >
                    {active && <span aria-hidden="true" className="-ml-2 h-4 w-0.5 shrink-0 rounded-full bg-accent" />}
                    <span className="min-w-0 flex-1 truncate">{c.title}</span>
                    <span className={`shrink-0 text-[11px] text-subtle ${active ? '' : 'opacity-0 group-hover:opacity-100'}`}>
                      {relativeDate(c.created_at)}
                    </span>
                  </button>
                  <button
                    aria-label={`Delete ${c.title}`}
                    title="Delete chat"
                    onClick={() => {
                      void (async () => {
                        try {
                          await api.deleteConversation(c.id)
                          setConversations((prev) => prev?.filter((p) => p.id !== c.id) ?? prev)
                          toast('Conversation deleted', 'success')
                          if (c.id === activeConvID) navigate('/')
                        } catch {
                          toast('Could not delete the conversation', 'error')
                        }
                      })()
                    }}
                    className={`shrink-0 rounded-md p-1.5 text-subtle transition hover:bg-raised hover:text-red-500 ${active ? '' : 'opacity-0 group-hover:opacity-100'}`}
                  >
                    <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                </li>
              )
            })}
          </ul>
        </li>
      ))}
    </ul>
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
    ['Older', olderItems],
  ]
  return buckets.filter(([, items]) => items.length > 0)
}

// relativeDate renders "3 days ago"-style labels with a plain-date fallback.
const rtf = typeof Intl !== 'undefined' && 'RelativeTimeFormat' in Intl
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

export function Sidebar({ openPath, onOpenNote, health, loading }: {
  openPath: string | null
  onOpenNote: (path: string) => void
  health: Health | null
  loading: boolean
}) {
  return (
    <aside className="flex w-72 shrink-0 flex-col border-r border-line bg-app">
      <div className="flex h-14 shrink-0 items-center gap-2.5 px-4">
        <span className="text-xl leading-none">🦉</span>
        <span className="font-display text-lg font-semibold tracking-tight text-heading">Thoth</span>
        <span className="ml-auto h-2 w-2 rounded-full bg-accent" aria-hidden="true" />
      </div>
      <div className="flex min-h-0 flex-1 flex-col">
        <div className="flex shrink-0 items-center justify-between px-3 pb-1">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-subtle">Chats</span>
          <button onClick={() => navigate('/')}
            className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-accent transition hover:bg-accent-soft">
            <Plus className="h-3.5 w-3.5" aria-hidden="true" />
            New chat
          </button>
        </div>
        <div className="min-h-0 flex-1">
          <ChatsList />
        </div>
      </div>
      <div className="flex min-h-0 flex-1 flex-col border-t border-line px-3">
        <div className="px-0 pb-1.5 pt-4 text-[11px] font-semibold uppercase tracking-wider text-subtle">
          Wiki
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto pb-3">
          <SearchPanel onOpen={onOpenNote} />
          <div className="mt-3">
            <WikiTree openPath={openPath} onOpenNote={onOpenNote} />
          </div>
        </div>
      </div>
      <HealthFooter health={health} loading={loading} />
    </aside>
  )
}

// HealthFooter is the bottom status bar: a health dot with a one-line
// reason, and the app version on the right.
function HealthFooter({ health, loading }: { health: Health | null; loading: boolean }) {
  let dot = 'bg-subtle animate-pulse'
  let label = 'Checking…'
  if (!loading) {
    if (health && health.claude.found && health.wiki.exists) {
      dot = 'bg-emerald-500'
      label = 'All systems go'
    } else if (health && !health.claude.found) {
      dot = 'bg-red-500'
      label = 'Claude CLI missing'
    } else if (health && !health.wiki.exists) {
      dot = 'bg-red-500'
      label = 'Wiki missing'
    } else {
      dot = 'bg-red-500'
      label = 'Server unreachable'
    }
  }
  return (
    <footer className="flex h-8 shrink-0 items-center justify-between border-t border-line px-3 text-[11px] text-subtle">
      <span className="flex min-w-0 items-center gap-1.5">
        <span aria-hidden="true" className={`h-1.5 w-1.5 shrink-0 rounded-full ${dot}`} />
        <span className="truncate">{label}</span>
      </span>
      <span className="shrink-0">v{health?.version ?? '…'}</span>
    </footer>
  )
}
