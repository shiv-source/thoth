import { useState } from 'react'
import { useSearch } from '../hooks/useSearch'
import { NoteViewer } from './NoteViewer'

export function SearchPanel() {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState<string | null>(null)
  const { results, loading } = useSearch(query)

  return (
    <div className="space-y-2">
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search your wiki…"
        className="w-full rounded-lg border border-paper-300 bg-white px-3 py-2 text-sm outline-none placeholder:text-ink-400 focus:border-accent-500 dark:border-night-700 dark:bg-night-900"
      />
      {loading && <p className="px-1 text-xs text-ink-500">Searching…</p>}
      {!loading && query && results.length === 0 && (
        <p className="px-1 text-xs text-ink-500">No notes match.</p>
      )}
      <ul className="space-y-1">
        {results.map((r) => (
          <li key={r.path}>
            <button onClick={() => setOpen(r.path)}
              className="w-full rounded-lg px-2 py-2 text-left transition hover:bg-paper-200 dark:hover:bg-night-800">
              <div className="truncate text-sm font-medium text-ink-900 dark:text-paper-100">{r.title}</div>
              <div className="truncate text-xs text-ink-500">{r.path}</div>
              <div className="mt-0.5 line-clamp-2 text-xs text-ink-700 dark:text-ink-400"
                dangerouslySetInnerHTML={{ __html: r.snippet }} />
            </button>
          </li>
        ))}
      </ul>
      {open && <NoteViewer path={open} onClose={() => setOpen(null)} />}
    </div>
  )
}
