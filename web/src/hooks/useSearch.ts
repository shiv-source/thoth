import { useEffect, useRef, useState } from 'react'
import { api, type SearchResult } from '../api/client'

export function useSearch(query: string) {
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const seq = useRef(0)

  useEffect(() => {
    const q = query.trim()
    if (!q) { setResults([]); return }
    const id = ++seq.current
    setLoading(true)
    timer.current = setTimeout(() => {
      api.search(q)
        .then((r) => { if (id === seq.current) setResults(r.results) })
        .catch(() => { if (id === seq.current) setResults([]) })
        .finally(() => { if (id === seq.current) setLoading(false) })
    }, 300)
    return () => clearTimeout(timer.current)
  }, [query])

  return { results, loading }
}
