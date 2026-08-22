import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Button, Empty, Input, List } from 'antd'
import type { InputRef } from 'antd'
import { ClockCircleOutlined } from '@ant-design/icons'
import { useSearch } from '../../hooks/useSearch'
import { useViewRoute } from '../../hooks/useView'
import { clearSearchHistory, commitSearch, selectSearchActive, selectSearchHistory, setSearchActive } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'

export function SearchPanel({ onOpen }: { onOpen: (path: string) => void }) {
    // The query rides the URL (?q=) so it survives a reload; typing
    // replaceState's the query string (no history spam, no re-render loop).
    // The draft itself stays local — the URL is the source of truth.
    const { query: urlQuery } = useViewRoute()
    const [query, setQuery] = useState(() => urlQuery ?? '')
    const dispatch = useAppDispatch()
    const { results, loading } = useSearch(query)
    const active = useAppSelector(selectSearchActive)
    const history = useAppSelector(selectSearchHistory)
    const inputRef = useRef<InputRef>(null)

    // Back/forward (or any real popstate) re-syncs the query.
    useEffect(() => {
        setQuery(urlQuery ?? '')
    }, [urlQuery])

    // New results reset the keyboard highlight to the first item.
    useEffect(() => {
        dispatch(setSearchActive(0))
    }, [results, dispatch])

    const onQueryChange = (v: string) => {
        setQuery(v)
        window.history.replaceState(null, '', v ? `/search?q=${encodeURIComponent(v)}` : '/search')
    }

    // commit records a deliberate search — Enter, or opening a result. The
    // slice dedupes, caps, and persists.
    const commit = (v: string) => {
        if (v.trim()) dispatch(commitSearch(v))
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
            dispatch(setSearchActive(results.length ? Math.min(active + 1, results.length - 1) : 0))
        } else if (e.key === 'ArrowUp') {
            e.preventDefault()
            dispatch(setSearchActive(Math.max(active - 1, 0)))
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
            <Input.Search
                ref={inputRef}
                value={query}
                onChange={(e) => onQueryChange(e.target.value)}
                onKeyDown={onKeyDown}
                autoFocus
                allowClear
                size="large"
                loading={loading}
                placeholder="Search your wiki…"
            />
            {query && (
                <div className="mt-1.5">
                    {loading && <p className="px-1 text-xs text-subtle">Searching…</p>}
                    {!loading && results.length === 0 && (
                        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No notes match." className="py-4" />
                    )}
                    {results.length > 0 && (
                        <List
                            size="small"
                            dataSource={results}
                            renderItem={(r, i) => (
                                <List.Item
                                    onClick={() => open(r.path)}
                                    onMouseEnter={() => dispatch(setSearchActive(i))}
                                    className={`cursor-pointer rounded-lg px-2 ${i === active ? 'bg-accent-soft' : 'hover:bg-raised'}`}
                                >
                                    <List.Item.Meta
                                        title={<span className="text-sm font-medium">{r.title}</span>}
                                        description={
                                            <>
                                                <div className="truncate text-xs text-subtle">{r.path}</div>
                                                {/* Safe: the server escapes note text before building the
                                                    snippet; only its <mark> markers survive as real tags. */}
                                                <div
                                                    className="mt-0.5 line-clamp-2 text-xs"
                                                    dangerouslySetInnerHTML={{ __html: r.snippet }}
                                                />
                                            </>
                                        }
                                    />
                                </List.Item>
                            )}
                        />
                    )}
                </div>
            )}
            {!query && history.length > 0 && (
                <div className="mt-1.5">
                    <div className="flex items-center justify-between px-1">
                        <p className="text-xs font-medium uppercase tracking-wide text-subtle">Recent searches</p>
                        <Button type="link" size="small" onClick={() => dispatch(clearSearchHistory())}>
                            Clear
                        </Button>
                    </div>
                    <List
                        size="small"
                        dataSource={history}
                        renderItem={(h) => (
                            <List.Item className="p-0">
                                <Button
                                    type="text"
                                    block
                                    className="flex h-auto items-center justify-start py-1.5 text-left"
                                    icon={<ClockCircleOutlined aria-hidden="true" />}
                                    onClick={() => onQueryChange(h)}
                                >
                                    <span className="truncate">{h}</span>
                                </Button>
                            </List.Item>
                        )}
                    />
                </div>
            )}
        </div>
    )
}
