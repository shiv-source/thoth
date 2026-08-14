import { useCallback, useEffect, useState } from 'react'
import { api, type Health } from '../api/client'

// useHealth fetches GET /api/health (claude binary + wiki state) and exposes
// a recheck() for the setup screen's "Re-check" button.
export function useHealth() {
  const [health, setHealth] = useState<Health | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const recheck = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setHealth(await api.health())
    } catch (e) {
      setHealth(null)
      setError(e instanceof Error ? e.message : 'failed to reach the server')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void recheck()
  }, [recheck])

  return { health, loading, error, recheck }
}
