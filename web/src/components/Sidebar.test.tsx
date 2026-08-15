import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Sidebar } from './Sidebar'

const today = new Date()
const yesterday = new Date(Date.now() - 86400000)
const older = new Date(Date.now() - 40 * 86400000)
const iso = (d: Date) => d.toISOString()

const conversations = {
  conversations: [
    { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1', title: 'Today chat', created_at: iso(today) },
    { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2', title: 'Yesterday chat', created_at: iso(yesterday) },
    { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa3', title: 'Old chat', created_at: iso(older) },
  ],
}

function stubAPI(handlers: Record<string, () => Response>) {
  const fetchMock = vi.fn((url: string) => {
    const make = handlers[url]
    if (make) return Promise.resolve(make())
    return Promise.resolve(new Response('not found', { status: 404 }))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function renderSidebar() {
  return render(<Sidebar openPath={null} onOpenNote={() => {}} />)
}

describe('Sidebar chats section', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    window.history.pushState(null, '', '/')
  })

  it('groups the conversation history by day with dates on hover', async () => {
    stubAPI({
      '/api/conversations': () => new Response(JSON.stringify(conversations), { status: 200 }),
      '/api/wiki/tree': () => new Response(JSON.stringify({ nodes: [] }), { status: 200 }),
    })
    renderSidebar()

    expect(await screen.findByText('Today')).toBeInTheDocument()
    expect(screen.getByText('Yesterday')).toBeInTheDocument()
    expect(screen.getByText('Older')).toBeInTheDocument()
    expect(screen.getByText('Today chat')).toBeInTheDocument()
    expect(screen.getByText('Old chat')).toBeInTheDocument()
  })

  it('navigates to a conversation when its row is clicked', async () => {
    stubAPI({
      '/api/conversations': () => new Response(JSON.stringify(conversations), { status: 200 }),
      '/api/wiki/tree': () => new Response(JSON.stringify({ nodes: [] }), { status: 200 }),
    })
    renderSidebar()
    await userEvent.click(await screen.findByText('Today chat'))
    expect(window.location.pathname).toBe('/chat/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
  })

  it('navigates to the root when New chat is clicked', async () => {
    stubAPI({
      '/api/conversations': () => new Response(JSON.stringify(conversations), { status: 200 }),
      '/api/wiki/tree': () => new Response(JSON.stringify({ nodes: [] }), { status: 200 }),
    })
    renderSidebar()
    await userEvent.click(await screen.findByRole('button', { name: /New chat/ }))
    expect(window.location.pathname).toBe('/')
  })

  it('shows empty and error states', async () => {
    const fetchMock = stubAPI({
      '/api/conversations': () => new Response(JSON.stringify({ conversations: [] }), { status: 200 }),
      '/api/wiki/tree': () => new Response(JSON.stringify({ nodes: [] }), { status: 200 }),
    })
    const { unmount } = renderSidebar()
    expect(await screen.findByText(/No conversations yet/)).toBeInTheDocument()
    unmount()

    stubAPI({
      '/api/conversations': () => new Response('boom', { status: 500 }),
      '/api/wiki/tree': () => new Response(JSON.stringify({ nodes: [] }), { status: 200 }),
    })
    renderSidebar()
    expect(await screen.findByText('Could not load conversations')).toBeInTheDocument()
    void fetchMock
  })
})
