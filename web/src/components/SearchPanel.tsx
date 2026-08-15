import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Search } from 'lucide-react'
import { useSearch } from '../hooks/useSearch'
import { useViewRoute } from '../hooks/useView'
import { EmptyState } from './EmptyState'

export function SearchPanel({ onOpen }: { onOpen: (path: string) => void }) {
    const { segment } = useViewRoute()
    // The query rides the URL (#/search/<q>) so it survives a reload; typing
    // replaceState's the hash (no history spam, no re-render loop).
    const [query, setQuery] = useState(() => segment ?? '')
    const [active, setActive] = useState(0)
    const { results, loading } = useSearch(query)
    const inputRef = useRef<HTMLInputElement>(null)

    // Back/forward (or any real hashchange) re-syncs the query.
    useEffect(() => {
        setQuery(segment ?? '')
    }, [segment])

    useEffect(() => {
        setActive(0)
    }, [results])

    const onQueryChange = (v: string) => {
        setQuery(v)
        const encoded = v ? encodeURIComponent(v).replace(/%2F/gi, '/') : ''
        window.history.replaceState(null, '', v ? `/search/${encoded}` : '/search')
    }

    const open = (path: string) => {
        onOpen(path)
        setQuery('')
    }

    const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'ArrowDown') {
            e.preventDefault()
            setActive((a) => (results.length ? Math.min(a + 1, results.length - 1) : 0))
        } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            setActive((a) => Math.max(a - 1, 0))
        } else if (e.key === 'Enter') {
            const r = results[active]
            if (r) {
                e.preventDefault()
                open(r.path)
            }
        } else if (e.key === 'Escape') {
            setQuery('')
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
        </div>
    )
}
