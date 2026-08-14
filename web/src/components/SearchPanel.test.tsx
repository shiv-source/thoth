import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SearchPanel } from './SearchPanel'

describe('SearchPanel', () => {
  it('renders results for a query', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ results: [{ path: 'meetings/a.md', title: 'Standup', kind: 'meeting', snippet: '…<mark>deploy</mark>…' }] }),
      { status: 200 })))
    render(<SearchPanel />)
    await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
    await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
    vi.unstubAllGlobals()
  })
})
