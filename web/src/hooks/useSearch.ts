import { useEffect, useRef, useState } from 'react'
import { api, type SearchResult } from '../api/client'

export function useSearch(query: string) {
    const [results, setResults] = useState<SearchResult[]>([])
    const [loading, setLoading] = useState(false)
    const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
    const abort = useRef<AbortController | null>(null)
    const seq = useRef(0)

    useEffect(() => {
        const q = query.trim()
        if (!q) {
            // A cleared query invalidates everything in flight: the seq bump makes
            // any late response stale, the abort stops the request server-side.
            seq.current++
            abort.current?.abort()
            setResults([])
            setLoading(false)
            return
        }
        const id = ++seq.current
        setLoading(true)
        timer.current = setTimeout(() => {
            const controller = new AbortController()
            abort.current = controller
            api.search(q, controller.signal)
                .then((r) => {
                    if (id === seq.current) setResults(r.results)
                })
                .catch(() => {
                    if (id === seq.current) setResults([])
                })
                .finally(() => {
                    if (id === seq.current) setLoading(false)
                })
        }, 300)
        return () => {
            clearTimeout(timer.current)
            // Covers both the next keystroke and unmount: a superseded request is
            // cancelled instead of running to completion server-side.
            abort.current?.abort()
        }
    }, [query])

    return { results, loading }
}
