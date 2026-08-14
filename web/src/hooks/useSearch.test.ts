import { act, renderHook, waitFor } from '@testing-library/react'
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

  it('ignores stale responses from an older query', async () => {
    vi.useFakeTimers()
    let resolveFirst!: (r: Response) => void
    const fetchMock = vi.fn()
      .mockReturnValueOnce(new Promise<Response>((resolve) => { resolveFirst = resolve }))
      .mockResolvedValueOnce(new Response(
        JSON.stringify({ results: [{ path: 'new.md', title: 'New', kind: 'note', snippet: '…' }] }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const { result, rerender } = renderHook(({ q }) => useSearch(q), { initialProps: { q: 'first' } })
    act(() => { vi.advanceTimersByTime(300) }) // first query fires its request
    expect(fetchMock).toHaveBeenCalledTimes(1)
    rerender({ q: 'second' })
    act(() => { vi.advanceTimersByTime(300) }) // second query fires its request
    expect(fetchMock).toHaveBeenCalledTimes(2)
    await act(async () => { await Promise.resolve() }) // newest response lands
    expect(result.current.results).toEqual([expect.objectContaining({ path: 'new.md' })])

    // The older request resolves last: its payload must be dropped.
    await act(async () => {
      await Promise.resolve()
      resolveFirst(new Response(
        JSON.stringify({ results: [{ path: 'stale.md', title: 'Stale', kind: 'note', snippet: '…' }] }), { status: 200 }))
    })
    await act(async () => { await Promise.resolve() })
    expect(result.current.results).toEqual([expect.objectContaining({ path: 'new.md' })])

    vi.useRealTimers()
    vi.unstubAllGlobals()
  })
})
