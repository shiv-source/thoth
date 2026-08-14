import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useSearch } from './useSearch'

describe('useSearch', () => {
  it('returns results after the debounce', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ results: [{ path: 'a.md', title: 'A', kind: 'note', snippet: '…' }] }), { status: 200 })))
    const { result } = renderHook(() => useSearch('a'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.results[0]?.path).toBe('a.md')
    vi.unstubAllGlobals()
  })
})
