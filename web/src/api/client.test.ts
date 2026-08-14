import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from './client'

describe('api.search', () => {
  const fetchMock = vi.fn()
  beforeEach(() => { vi.stubGlobal('fetch', fetchMock) })
  afterEach(() => vi.unstubAllGlobals())

  it('parses results through zod', async () => {
    fetchMock.mockResolvedValue(new Response(
      JSON.stringify({ results: [{ path: 'meetings/a.md', title: 'A', kind: 'meeting', snippet: '…<mark>x</mark>…' }] }),
      { status: 200 }))
    const { results } = await api.search('x')
    expect(results[0]?.path).toBe('meetings/a.md')
  })

  it('rejects malformed payloads', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ results: [{ path: 42 }] }), { status: 200 }))
    await expect(api.search('x')).rejects.toThrow()
  })
})
