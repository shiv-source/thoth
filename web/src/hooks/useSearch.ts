import { useEffect, useRef } from 'react'
import { clearSearch, searchNotes, selectSearchLoading, selectSearchResults } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'

// useSearch debounces the query (300 ms) and dispatches it into the search
// slice with an AbortController, so a superseded request stops server-side
// and its late response is dropped by the slice's query guard. Clearing the
// query resets the slice immediately.
export function useSearch(query: string) {
    const dispatch = useAppDispatch()
    const results = useAppSelector(selectSearchResults)
    const loading = useAppSelector(selectSearchLoading)
    const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
    const abort = useRef<AbortController | null>(null)

    useEffect(() => {
        const q = query.trim()
        if (!q) {
            abort.current?.abort()
            dispatch(clearSearch())
            return
        }
        timer.current = setTimeout(() => {
            const controller = new AbortController()
            abort.current = controller
            void dispatch(searchNotes(q, { signal: controller.signal }))
        }, 300)
        return () => {
            clearTimeout(timer.current)
            // Covers both the next keystroke and unmount: a superseded
            // request is cancelled instead of running to completion.
            abort.current?.abort()
        }
    }, [query, dispatch])

    return { results: results ?? [], loading }
}
