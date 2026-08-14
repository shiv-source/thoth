import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ConversationMenu } from './ConversationMenu'

const conv = { id: 'c1', title: 'Tuesday standup', created_at: '2026-08-13T09:00:00Z' }
const messages = [
  { id: 1, conversation_id: 'c1', role: 'user', content: 'hi', created_at: '2026-08-13T09:00:00Z' },
  { id: 2, conversation_id: 'c1', role: 'assistant', content: 'hello', created_at: '2026-08-13T09:00:01Z' },
]

describe('ConversationMenu', () => {
  it('lists conversations and loads one on click', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ conversations: [conv] }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ conversation: conv, messages }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    const onSelect = vi.fn()

    render(<ConversationMenu onSelect={onSelect} />)
    await userEvent.click(screen.getByRole('button', { name: 'Conversation history' }))
    await userEvent.click(await screen.findByText('Tuesday standup'))

    expect(onSelect).toHaveBeenCalledWith([
      { role: 'user', content: 'hi' },
      { role: 'assistant', content: 'hello' },
    ], 'c1')
    expect(screen.queryByText('Tuesday standup')).not.toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it('shows the empty state when there are no conversations', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ conversations: [] }), { status: 200 })))
    render(<ConversationMenu onSelect={() => {}} />)
    await userEvent.click(screen.getByRole('button', { name: 'Conversation history' }))
    expect(await screen.findByText('No conversations yet')).toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it('closes on Esc', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ conversations: [conv] }), { status: 200 })))
    render(<ConversationMenu onSelect={() => {}} />)
    const button = screen.getByRole('button', { name: 'Conversation history' })
    await userEvent.click(button)
    await screen.findByText('Tuesday standup')

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByText('Tuesday standup')).not.toBeInTheDocument()
    vi.unstubAllGlobals()
  })
})
