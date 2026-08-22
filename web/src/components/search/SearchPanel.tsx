import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Input } from 'antd'
import type { InputRef } from 'antd'
import { useSearch } from '../../hooks/useSearch'
import { useViewRoute } from '../../hooks/useView'
import { clearSearchHistory, commitSearch, selectSearchActive, selectSearchHistory, setSearchActive } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { RecentSearches } from './RecentSearches'
import { SearchResults } from './SearchResults'

// SearchPanel is the wiki search surface: the query rides the URL (?q=) so
// it survives a reload, the results list renders via SearchResults, and the
// empty-query state shows RecentSearches. Keyboard navigation (arrows +
// Enter + Escape) walks the result list through the ui slice's active index.
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
            {query ? (
                <SearchResults
                    results={results}
                    loading={loading}
                    active={active}
                    onOpen={open}
                    onHover={(i) => dispatch(setSearchActive(i))}
                />
            ) : (
                <RecentSearches
                    history={history}
                    onPick={onQueryChange}
                    onClear={() => dispatch(clearSearchHistory())}
                />
            )}
        </div>
    )
}
