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

describe('api.getConversation and health', () => {
  const fetchMock = vi.fn()
  beforeEach(() => { vi.stubGlobal('fetch', fetchMock) })
  afterEach(() => vi.unstubAllGlobals())

  it('parses conversation history through zod', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({
      conversation: { id: 'c1', title: 'T', created_at: '2026-08-13T09:00:00Z' },
      messages: [{ id: 1, conversation_id: 'c1', role: 'user', content: 'hi', created_at: '2026-08-13T09:00:00Z' }],
    }), { status: 200 }))
    const { messages, conversation } = await api.getConversation('c1')
    expect(conversation.id).toBe('c1')
    expect(messages[0]?.content).toBe('hi')
    expect(fetchMock).toHaveBeenCalledWith('/api/conversations/c1')
  })

  it('rejects unknown message roles', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({
      conversation: { id: 'c1', title: 'T', created_at: '2026-08-13T09:00:00Z' },
      messages: [{ id: 1, conversation_id: 'c1', role: 'system', content: 'hi', created_at: '2026-08-13T09:00:00Z' }],
    }), { status: 200 }))
    await expect(api.getConversation('c1')).rejects.toThrow()
  })

  it('parses health and doctor responses', async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({
      status: 'ok',
      claude: { found: true, path: '/usr/local/bin/claude' },
      wiki: { path: '/tmp/wiki', exists: true },
      version: '1.2.3',
    }), { status: 200 }))
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({
      checks: [{ name: 'config', ok: true, message: 'parses' }],
    }), { status: 200 }))
    const health = await api.health()
    expect(health.claude.found).toBe(true)
    const { checks } = await api.doctor()
    expect(checks[0]?.name).toBe('config')
  })
})
