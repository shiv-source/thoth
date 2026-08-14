import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SettingsPanel } from './SettingsPanel'

const settings = { wiki_path: '~/.thoth/wiki', host: '127.0.0.1', port: 8333, claude_bin: '', permission_mode: '', model: '' }

describe('SettingsPanel', () => {
  it('loads current settings and saves edits', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(settings), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...settings, port: 9444 }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    render(<SettingsPanel />)
    const port = await screen.findByDisplayValue('8333')
    await userEvent.clear(port)
    await userEvent.type(port, '9444')
    await userEvent.click(screen.getByRole('button', { name: /Save settings/ }))
    await waitFor(() => expect(screen.getByText(/Saved/)).toBeInTheDocument())
    vi.unstubAllGlobals()
  })
})
