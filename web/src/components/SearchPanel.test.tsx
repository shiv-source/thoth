import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SearchPanel } from './SearchPanel'

function stubSearch() {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
    JSON.stringify({ results: [{ path: 'meetings/a.md', title: 'Standup', kind: 'meeting', snippet: '…<mark>deploy</mark>…' }] }),
    { status: 200 })))
}

describe('SearchPanel', () => {
  it('renders results for a query', async () => {
    stubSearch()
    render(<SearchPanel onOpen={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
    await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
    vi.unstubAllGlobals()
  })

  it('opens the highlighted note on Enter', async () => {
    stubSearch()
    const onOpen = vi.fn()
    render(<SearchPanel onOpen={onOpen} />)
    await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
    await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
    await userEvent.keyboard('{Enter}')
    expect(onOpen).toHaveBeenCalledWith('meetings/a.md')
    vi.unstubAllGlobals()
  })

  it('clears the query on Escape', async () => {
    stubSearch()
    render(<SearchPanel onOpen={() => {}} />)
    const input = screen.getByPlaceholderText(/Search your wiki/)
    await userEvent.type(input, 'deploy')
    await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
    await userEvent.keyboard('{Escape}')
    expect(input).toHaveValue('')
    expect(screen.queryByText('Standup')).not.toBeInTheDocument()
    vi.unstubAllGlobals()
  })
})
