import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SettingsModal } from './SettingsModal'
import { ToastProvider } from './Toast'

const settings = {
  wiki_path: '~/.thoth/wiki', host: '127.0.0.1', port: 8333,
  claude_bin: '', permission_mode: '', model: '', repo_url: '',
}

const emptyGitHub = {
  username: '', display_name: '', email: '', avatar_url: '', profile_url: '', scopes: '', account_created_at: '', account_updated_at: '',
}

const connected = {
  username: 'octo', display_name: 'Octo Cat', email: 'octo@example.com', avatar_url: '',
  profile_url: 'https://github.com/octo', scopes: 'repo,user',
  account_created_at: '2018-05-01T10:00:00Z', account_updated_at: '2026-08-01T10:00:00Z',
}

// stubAPI answers fetches by "METHOD path" with canned responses and returns
// the mock so tests can assert on the calls.
function stubAPI(handlers: Record<string, () => Response>) {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${url}`
    const make = handlers[key]
    if (make) return Promise.resolve(make())
    return Promise.resolve(new Response('not found', { status: 404 }))
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const getSettings = () => new Response(JSON.stringify(settings), { status: 200 })
const getEmptyGitHub = () => new Response(JSON.stringify(emptyGitHub), { status: 200 })

function renderModal() {
  return render(<ToastProvider><SettingsModal onClose={() => {}} /></ToastProvider>)
}

afterEach(() => vi.unstubAllGlobals())

describe('SettingsModal', () => {
  it('loads current settings and saves edits', async () => {
    stubAPI({
      'GET /api/settings': getSettings,
      'GET /api/github/auth': getEmptyGitHub,
      'PUT /api/settings': () => new Response(JSON.stringify({ ...settings, port: 9444 }), { status: 200 }),
    })

    renderModal()
    const port = await screen.findByDisplayValue('8333')
    await userEvent.clear(port)
    await userEvent.type(port, '9444')
    await userEvent.click(screen.getByRole('button', { name: /Save/ }))
    await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
    // The save also surfaces as a toast.
    expect(await screen.findByText('Settings saved')).toBeInTheDocument()
  })

  it('keeps the dialog at a fixed width and height', async () => {
    stubAPI({ 'GET /api/settings': getSettings, 'GET /api/github/auth': getEmptyGitHub })
    renderModal()
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveClass('h-[36rem]', 'w-[48rem]')
  })

  it('closes on escape and on backdrop click', async () => {
    stubAPI({ 'GET /api/settings': getSettings, 'GET /api/github/auth': getEmptyGitHub })
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
  })

  it('runs the doctor checks in the Doctor tab', async () => {
    stubAPI({
      'GET /api/settings': getSettings,
      'GET /api/github/auth': getEmptyGitHub,
      'GET /api/doctor': () => new Response(JSON.stringify({
        checks: [
          { name: 'config', ok: true, message: '/tmp/config.toml parses' },
          { name: 'claude', ok: false, message: 'claude CLI not found on PATH' },
        ],
      }), { status: 200 }),
    })

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Doctor' }))
    expect(await screen.findByText('config')).toBeInTheDocument()
    expect(screen.getByText('claude CLI not found on PATH')).toBeInTheDocument()
  })

  it('stores the git remote URL and pushes', async () => {
    const fetchMock = stubAPI({
      'GET /api/settings': getSettings,
      'GET /api/github/auth': () => new Response(JSON.stringify(connected), { status: 200 }),
      'POST /api/git/setup': () => new Response(JSON.stringify({ ok: true }), { status: 200 }),
    })

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
    const url = await screen.findByPlaceholderText(/github\.com/)
    await userEvent.type(url, 'https://example.com/wiki.git')
    await userEvent.click(screen.getByRole('button', { name: 'Initialize & Push' }))
    expect(await screen.findByText('Wiki pushed to remote')).toBeInTheDocument()
    // The setup call carried the URL.
    const gitBody = fetchMock.mock.calls.find(([u]) => u === '/api/git/setup')?.[1]?.body
    expect(typeof gitBody === 'string' && gitBody.includes('https://example.com/wiki.git')).toBe(true)
  })

  it('connects a GitHub account with a token', async () => {
    const fetchMock = stubAPI({
      'GET /api/settings': getSettings,
      'GET /api/github/auth': getEmptyGitHub,
      'POST /api/github/auth': () => new Response(JSON.stringify(connected), { status: 200 }),
    })

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
    await userEvent.type(await screen.findByPlaceholderText(/ghp_/), 'ghp_secret123')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
    expect(screen.getByText('GitHub connected')).toBeInTheDocument()
    // The stored account facts render view-only.
    expect(screen.getByText(/Member since 2018-05-01/)).toBeInTheDocument()
    // The remote URL input appears once connected.
    expect(await screen.findByPlaceholderText(/github\.com/)).toBeInTheDocument()
    // The POST carried the token.
    const connectBody = fetchMock.mock.calls.find(
      ([u, init]) => u === '/api/github/auth' && init?.method === 'POST',
    )?.[1]?.body
    expect(typeof connectBody === 'string' && connectBody.includes('ghp_secret123')).toBe(true)
  })

  it('shows the connect error', async () => {
    stubAPI({
      'GET /api/settings': getSettings,
      'GET /api/github/auth': getEmptyGitHub,
      'POST /api/github/auth': () => new Response(JSON.stringify({ error: 'github rejected the token' }), { status: 400 }),
    })

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
    await userEvent.type(await screen.findByPlaceholderText(/ghp_/), 'bad-token')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    // The message renders inline and as a toast.
    expect((await screen.findAllByText('github rejected the token')).length).toBeGreaterThan(0)
  })

  it('disconnects a GitHub account', async () => {
    const fetchMock = stubAPI({
      'GET /api/settings': getSettings,
      'GET /api/github/auth': () => new Response(JSON.stringify(connected), { status: 200 }),
      'DELETE /api/github/auth': () => new Response(JSON.stringify({ ok: true }), { status: 200 }),
    })

    renderModal()
    await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
    expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Disconnect' }))

    expect(await screen.findByPlaceholderText(/ghp_/)).toBeInTheDocument()
    expect(screen.getByText('GitHub disconnected')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u, init]) => u === '/api/github/auth' && init?.method === 'DELETE')).toBe(true)
  })
})
