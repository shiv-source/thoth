import { useEffect, useState } from 'react'
import { api, type Conversation } from '../api/client'
import type { ChatMessage } from '../hooks/useChat'

// The Intl.RelativeTimeFormat formatter (with the plain-date fallback).
const rtf = typeof Intl !== 'undefined' && 'RelativeTimeFormat' in Intl
  ? new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
  : null

/** "3 days ago"-style label for a conversation's created_at, falling back to
 *  a plain date when the timestamp is unparseable or the API is unavailable. */
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

export function ConversationMenu({ onSelect }: {
  onSelect: (messages: ChatMessage[], conversationId: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [conversations, setConversations] = useState<Conversation[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Esc closes; the fixed backdrop below closes on outside clicks.
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  const toggle = async () => {
    if (!open) {
      try {
        const res = await api.listConversations()
        setConversations(res.conversations)
        setError(null)
      } catch {
        setError('Could not load conversations')
      }
    }
    setOpen((o) => !o)
  }

  const pick = async (id: string) => {
    setOpen(false)
    try {
      const res = await api.getConversation(id)
      onSelect(res.messages.map((m) => ({ role: m.role, content: m.content })), id)
    } catch {
      setError('Could not load conversation')
    }
  }

  return (
    <div className="relative">
      <button onClick={() => void toggle()} aria-label="Conversation history"
        aria-expanded={open} title="Conversation history"
        className="rounded-lg p-2 text-subtle transition hover:bg-raised hover:text-ink">
        <ClockIcon />
      </button>
      {open && (
        <>
          {/* Backdrop: any click outside the card closes the menu. */}
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} aria-hidden="true" />
          <div className="absolute right-0 top-full z-50 mt-1 w-72 overflow-hidden rounded-xl border border-line bg-surface shadow-md">
            <div className="border-b border-line px-4 py-2.5 text-xs font-semibold uppercase tracking-wider text-subtle">
              History
            </div>
            <ul className="max-h-80 overflow-y-auto p-1.5">
              {error && <li className="px-2 py-1.5 text-sm text-red-600 dark:text-red-400">{error}</li>}
              {conversations && conversations.length === 0 && (
                <li className="px-2 py-1.5 text-sm text-subtle">No conversations yet</li>
              )}
              {conversations?.map((c) => (
                <li key={c.id}>
                  <button onClick={() => void pick(c.id)}
                    className="flex w-full items-baseline justify-between gap-3 rounded-lg px-2 py-1.5 text-left text-sm transition hover:bg-raised">
                    <span className="min-w-0 truncate text-ink">{c.title}</span>
                    <span className="shrink-0 text-xs text-subtle">{relativeDate(c.created_at)}</span>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </>
      )}
    </div>
  )
}

function ClockIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"
      className="h-4 w-4">
      <circle cx="10" cy="10" r="7.5" />
      <path d="M10 5.5V10l3 2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
