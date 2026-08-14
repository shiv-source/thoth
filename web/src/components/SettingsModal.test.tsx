import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SettingsModal } from './SettingsModal'
import { ToastProvider } from './Toast'

const settings = {
  wiki_path: '~/.thoth/wiki', host: '127.0.0.1', port: 8333,
  claude_bin: '', permission_mode: '', model: '', git_remote_url: '',
}

function renderModal() {
  return render(<ToastProvider><SettingsModal onClose={() => {}} /></ToastProvider>)
}

describe('SettingsModal', () => {
  it('loads current settings and saves edits', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(settings), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...settings, port: 9444 }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    renderModal()
    const port = await screen.findByDisplayValue('8333')
    await userEvent.clear(port)
    await userEvent.type(port, '9444')
    await userEvent.click(screen.getByRole('button', { name: /Save/ }))
    await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
    // The save also surfaces as a toast.
    expect(await screen.findByText('Settings saved')).toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it('closes on escape and on backdrop click', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(settings), { status: 200 })))
    const onClose = vi.fn()
    const { container } = render(<ToastProvider><SettingsModal onClose={onClose} /></ToastProvider>)
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

  it('runs the doctor checks in the Doctor tab', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(settings), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        checks: [
          { name: 'config', ok: true, message: '/tmp/config.toml parses' },
          { name: 'claude', ok: false, message: 'claude CLI not found on PATH' },
        ],
      }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Doctor' }))
    expect(await screen.findByText('config')).toBeInTheDocument()
    expect(screen.getByText('claude CLI not found on PATH')).toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it('stores the git remote URL and pushes', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(settings), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
    const url = await screen.findByPlaceholderText(/github\.com/)
    await userEvent.type(url, 'https://example.com/wiki.git')
    await userEvent.click(screen.getByRole('button', { name: 'Initialize & Push' }))
    expect(await screen.findByText('Wiki pushed to remote')).toBeInTheDocument()
    // The setup call carried the URL.
    const gitCall = fetchMock.mock.calls.find(([u]) => u === '/api/git/setup')
    expect(gitCall).toBeDefined()
    vi.unstubAllGlobals()
  })
})
