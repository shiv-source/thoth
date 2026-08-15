import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Clock, Search } from 'lucide-react'
import { useSearch } from '../hooks/useSearch'
import { useViewRoute } from '../hooks/useView'
import { clearSearchHistory, commitSearch, selectSearchHistory } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { EmptyState } from './EmptyState'

export function SearchPanel({ onOpen }: { onOpen: (path: string) => void }) {
    // The query rides the URL (?q=) so it survives a reload; typing
    // replaceState's the query string (no history spam, no re-render loop).
    const { query: urlQuery } = useViewRoute()
    const [query, setQuery] = useState(() => urlQuery ?? '')
    const [active, setActive] = useState(0)
    const { results, loading } = useSearch(query)
    const history = useAppSelector(selectSearchHistory)
    const dispatch = useAppDispatch()
    const inputRef = useRef<HTMLInputElement>(null)

    // Back/forward (or any real popstate) re-syncs the query.
    useEffect(() => {
        setQuery(urlQuery ?? '')
    }, [urlQuery])

    useEffect(() => {
        setActive(0)
    }, [results])

    const onQueryChange = (v: string) => {
        setQuery(v)
        window.history.replaceState(null, '', v ? `/search?q=${encodeURIComponent(v)}` : '/search')
    }

    // commit records a deliberate search — Enter, or opening a result. The
    // slice dedupes, caps, and persists.
    const commit = (v: string) => {
        if (v.trim()) dispatch(commitSearch(v))
    }

    const clearHistory = () => {
        dispatch(clearSearchHistory())
    }

    const open = (path: string) => {
        commit(query)
        // Clear the search URL first: navigating to the note (onOpen) pushes
        // /notes/<path>, and this replaceState would clobber it afterwards.
        onQueryChange('')
        onOpen(path)
    }

    const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'ArrowDown') {
            e.preventDefault()
            setActive((a) => (results.length ? Math.min(a + 1, results.length - 1) : 0))
        } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setActive((a) => Math.max(a - 1, 0))
        } else if (e.key === 'Enter') {
            commit(query)
            const r = results[active]
            if (r) {
                e.preventDefault()
                open(r.path)
            }
        } else if (e.key === 'Escape') {
            onQueryChange('')
            inputRef.current?.blur()
        }
    }

    return (
        <div>
            <div className="relative">
                <Search
                    aria-hidden="true"
                    className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-subtle"
                />
                <input
                    ref={inputRef}
                    value={query}
                    onChange={(e) => onQueryChange(e.target.value)}
                    onKeyDown={onKeyDown}
                    autoFocus
                    placeholder="Search your wiki…"
                    className="w-full rounded-lg border border-line bg-surface py-2 pl-8 pr-3 text-sm text-ink outline-none placeholder:text-subtle/70 focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500"
                />
            </div>
            {query && (
                <div className="mt-1.5">
                    {loading && <p className="px-1 text-xs text-subtle">Searching…</p>}
                    {!loading && results.length === 0 && (
                        <EmptyState icon="🔍" title="No notes match." className="py-4 text-xs" />
                    )}
                    {results.length > 0 && (
                        <ul className="space-y-0.5">
                            {results.map((r, i) => (
                                <li key={r.path}>
                                    <button
                                        onClick={() => open(r.path)}
                                        onMouseEnter={() => setActive(i)}
                                        className={`w-full rounded-lg px-2 py-1.5 text-left transition ${
                                            i === active ? 'bg-accent-soft' : 'hover:bg-raised'
                                        }`}
                                    >
                                        <div className="truncate text-sm font-medium text-ink">{r.title}</div>
                                        <div className="truncate text-xs text-subtle">{r.path}</div>
                                        {/* Safe: the server escapes note text before building the
                        snippet; only its <mark> markers survive as real tags. */}
                                        <div
                                            className="mt-0.5 line-clamp-2 text-xs text-ink/80"
                                            dangerouslySetInnerHTML={{ __html: r.snippet }}
                                        />
                                    </button>
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            )}
            {!query && history.length > 0 && (
                <div className="mt-1.5">
                    <div className="flex items-center justify-between px-1">
                        <p className="text-xs font-medium uppercase tracking-wide text-subtle">Recent searches</p>
                        <button onClick={clearHistory} className="text-xs text-subtle transition hover:text-ink">
                            Clear
                        </button>
                    </div>
                    <ul className="mt-1 space-y-0.5">
                        {history.map((h) => (
                            <li key={h}>
                                <button
                                    onClick={() => onQueryChange(h)}
                                    className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm text-ink transition hover:bg-raised"
                                >
                                    <Clock className="h-3.5 w-3.5 shrink-0 text-subtle" aria-hidden="true" />
                                    <span className="truncate">{h}</span>
                                </button>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
        </div>
    )
}
