import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SettingsModal } from './SettingsModal'

const settings = { wiki_path: '~/.thoth/wiki', host: '127.0.0.1', port: 8333, claude_bin: '', permission_mode: '', model: '' }

describe('SettingsModal', () => {
  it('loads current settings and saves edits', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(settings), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...settings, port: 9444 }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    render(<SettingsModal onClose={() => {}} />)
    const port = await screen.findByDisplayValue('8333')
    await userEvent.clear(port)
    await userEvent.type(port, '9444')
    await userEvent.click(screen.getByRole('button', { name: /Save/ }))
    await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
    vi.unstubAllGlobals()
  })

  it('closes on escape and on backdrop click', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(settings), { status: 200 })))
    const onClose = vi.fn()
    const { container } = render(<SettingsModal onClose={onClose} />)
    await screen.findByDisplayValue('8333')

    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)

    // The backdrop is the modal's root element; a click on it must close,
    // while a click inside the dialog card must not.
    await userEvent.click(container.firstElementChild!)
    expect(onClose).toHaveBeenCalledTimes(2)

    await userEvent.click(screen.getByRole('dialog'))
    expect(onClose).toHaveBeenCalledTimes(2)
    vi.unstubAllGlobals()
  })
})
