import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SearchPanel } from './SearchPanel'
import { axiosModuleMock } from '../test/mockAxios'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

function stubSearch() {
  mocks.get.mockResolvedValue({
    data: { results: [{ path: 'meetings/a.md', title: 'Standup', kind: 'meeting', snippet: '…<mark>deploy</mark>…' }] },
  })
}

describe('SearchPanel', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('renders results for a query', async () => {
    stubSearch()
    render(<SearchPanel onOpen={() => {}} />)
    await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
    await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
  })

  it('opens the highlighted note on Enter', async () => {
    stubSearch()
    const onOpen = vi.fn()
    render(<SearchPanel onOpen={onOpen} />)
    await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
    await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
    await userEvent.keyboard('{Enter}')
    expect(onOpen).toHaveBeenCalledWith('meetings/a.md')
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
  })
})
