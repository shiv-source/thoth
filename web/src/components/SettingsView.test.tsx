import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SettingsView } from './SettingsView'
import { ToastProvider } from './Toast'
import { renderWithStore } from '../test/renderWithStore'

// The client creates its axios instance via axios.create; the mocks are
// hoisted so the (also hoisted) vi.mock factory can close over them.
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))

vi.mock('axios', () => ({
    default: {
        create: () => ({
            get: mocks.get,
            post: mocks.post,
            put: mocks.put,
            delete: mocks.delete
        }),
        isAxiosError: (e: unknown) =>
            !!(e && typeof e === 'object' && (e as { isAxiosError?: boolean }).isAxiosError === true)
    }
}))

// axiosError builds a rejection value shaped like an axios error response.
function axiosError(status: number, body: unknown) {
    return Object.assign(new Error(`${status}`), {
        isAxiosError: true,
        response: { status, statusText: String(status), data: body }
    })
}

// stubAPI wires the mocks to the handlers keyed by "METHOD /path"; handlers
// return the response BODY directly — axios wraps it as `{ data }`, which
// the client parses via zod.
function stubAPI(handlers: Record<string, () => unknown>) {
    const respond = (method: string, url: string) => {
        const make = handlers[`${method} ${url}`] ?? handlers[url]
        if (!make) {
            return Promise.reject(
                Object.assign(new Error(`unhandled ${method} ${url}`), {
                    isAxiosError: true,
                    response: { status: 500, statusText: 'Internal Server Error' }
                })
            )
        }
        return Promise.resolve({ data: make() })
    }
    mocks.get.mockImplementation((url: string) => respond('GET', url))
    mocks.post.mockImplementation((url: string) => respond('POST', url))
    mocks.put.mockImplementation((url: string) => respond('PUT', url))
    mocks.delete.mockImplementation((url: string) => respond('DELETE', url))
    return mocks
}

// Body of the most recent call to `method url` (calls accumulate across
// tests; mock call args are `any`, so narrow them through `unknown`).
function lastBody(method: 'get' | 'post' | 'put' | 'delete', url: string): unknown {
    return [...mocks[method].mock.calls].reverse().find(([u]) => u === url)?.[1] as unknown
}

const settings = {
    wiki_path: '~/.thoth/wiki',
    model: '',
    repo_url: '',
    sync_enabled: false
}

const emptyGitHub = {
    username: '',
    display_name: '',
    email: '',
    avatar_url: '',
    profile_url: '',
    scopes: '',
    account_created_at: '',
    account_updated_at: ''
}

const connected = {
    username: 'octo',
    display_name: 'Octo Cat',
    email: 'octo@example.com',
    avatar_url: '',
    profile_url: 'https://github.com/octo',
    scopes: 'repo,user',
    account_created_at: '2018-05-01T10:00:00Z',
    account_updated_at: '2026-08-01T10:00:00Z'
}

const getSettings = () => settings
const getEmptyGitHub = () => emptyGitHub
const getRepos = () => ({
    repos: [
        {
            full_name: 'octo/wiki',
            clone_url: 'https://github.com/octo/wiki.git',
            private: true,
            description: 'My personal knowledge base'
        },
        {
            full_name: 'octo/public-wiki',
            clone_url: 'https://github.com/octo/public-wiki.git',
            private: false,
            description: ''
        }
    ]
})

function renderSettings() {
    return renderWithStore(
        <ToastProvider>
            <SettingsView />
        </ToastProvider>
    )
}

describe('SettingsView', () => {
    it('loads current settings and saves edits', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': getEmptyGitHub,
            'PUT /api/settings': () => ({ ...settings, wiki_path: '/tmp/other/wiki' })
        })

        renderSettings()
        const wikiPath = await screen.findByDisplayValue('~/.thoth/wiki')
        await userEvent.clear(wikiPath)
        await userEvent.type(wikiPath, '/tmp/other/wiki')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
        // The save also surfaces as a toast.
        expect(await screen.findByText('Settings saved')).toBeInTheDocument()
    })

    it('selects a model from the models endpoint and saves it', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': getEmptyGitHub,
            'GET /api/models': () => ({
                models: [
                    { value: '', label: 'CLI default', provider: 'Claude Code' },
                    { value: 'deepseek-v4-flash', label: 'V4 Flash — fastest', provider: 'DeepSeek' }
                ]
            }),
            'PUT /api/settings': () => ({ ...settings })
        })

        renderSettings()
        const select = await screen.findByRole('combobox')
        await userEvent.selectOptions(select, 'deepseek-v4-flash')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
        expect(lastBody('put', '/api/settings')).toMatchObject({ model: 'deepseek-v4-flash' })
    })

    it('shows the save error when the server rejects', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': getEmptyGitHub
        })
        mocks.put.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))

        renderSettings()
        const wikiPath = await screen.findByDisplayValue('~/.thoth/wiki')
        await userEvent.clear(wikiPath)
        await userEvent.type(wikiPath, '/tmp/other/wiki')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))

        // The message renders inline (with a period) and as a toast.
        expect((await screen.findAllByText(/Could not save settings/)).length).toBeGreaterThan(0)
    })

    it('runs the doctor checks in the Doctor tab', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': getEmptyGitHub,
            'GET /api/doctor': () => ({
                checks: [
                    { name: 'wiki', ok: true, message: '/tmp/wiki exists' },
                    { name: 'claude', ok: false, message: 'claude CLI not found on PATH' }
                ]
            })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Doctor' }))
        expect(await screen.findByText('wiki')).toBeInTheDocument()
        expect(screen.getByText('claude CLI not found on PATH')).toBeInTheDocument()
    })

    it('stores the git remote URL and pushes', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': () => connected,
            'GET /api/github/repos': getRepos,
            'POST /api/git/setup': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
        // The connected account's repos are offered as a dropdown the width of
        // the input; picking the private repo fills the field.
        const url = await screen.findByPlaceholderText(/github\.com/)
        await userEvent.type(url, 'octo')
        const option = await screen.findByRole('button', { name: /octo\/wiki/ })
        expect(await screen.findByText('My personal knowledge base')).toBeInTheDocument()
        await userEvent.click(option)
        expect(url).toHaveValue('https://github.com/octo/wiki.git')
        await userEvent.click(screen.getByRole('button', { name: 'Initialize & Push' }))
        expect(await screen.findByText('Wiki pushed to remote')).toBeInTheDocument()
        // The setup call carried the URL.
        const gitBody = lastBody('post', '/api/git/setup')
        expect(gitBody != null && JSON.stringify(gitBody).includes('https://github.com/octo/wiki.git')).toBe(true)
    })

    it('connects a GitHub account with a token', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': getEmptyGitHub,
            'GET /api/github/repos': getRepos,
            'POST /api/github/auth': () => connected
        })

        renderSettings()
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
        const connectBody = lastBody('post', '/api/github/auth')
        expect(connectBody != null && JSON.stringify(connectBody).includes('ghp_secret123')).toBe(true)
    })

    it('shows the connect error', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': getEmptyGitHub
        })
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'github rejected the token' }))

        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
        await userEvent.type(await screen.findByPlaceholderText(/ghp_/), 'bad-token')
        await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

        // The message renders inline and as a toast.
        expect((await screen.findAllByText('github rejected the token')).length).toBeGreaterThan(0)
    })

    it('disconnects a GitHub account', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': () => connected,
            'GET /api/github/repos': getRepos,
            'DELETE /api/github/auth': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
        expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
        await userEvent.click(screen.getByRole('button', { name: 'Disconnect' }))

        expect(await screen.findByPlaceholderText(/ghp_/)).toBeInTheDocument()
        expect(screen.getByText('GitHub disconnected')).toBeInTheDocument()
        expect(mocks.delete.mock.calls.some(([u]) => u === '/api/github/auth')).toBe(true)
    })
})

describe('SettingsView auto-sync', () => {
    it('toggles sync_enabled in the Git tab and saves it', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': () => connected,
            'GET /api/github/repos': getRepos,
            'PUT /api/settings': () => ({ ...settings, sync_enabled: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
        await userEvent.click(await screen.findByRole('checkbox'))
        await userEvent.click(screen.getByRole('button', { name: 'Save' }))

        expect(await screen.findByText('Settings saved')).toBeInTheDocument()
        const put = lastBody('put', '/api/settings')
        expect(put != null && JSON.stringify(put).includes('"sync_enabled":true')).toBe(true)
    })
})

describe('SettingsView public repo guard', () => {
    it('warns and blocks push when a public repo is picked', async () => {
        stubAPI({
            'GET /api/settings': getSettings,
            'GET /api/github/auth': () => connected,
            'GET /api/github/repos': getRepos
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
        const url = await screen.findByPlaceholderText(/github\.com/)
        await userEvent.type(url, 'octo/public')
        await userEvent.click(await screen.findByRole('button', { name: /octo\/public-wiki/ }))

        expect(url).toHaveValue('https://github.com/octo/public-wiki.git')
        expect(await screen.findByText(/public repository is blocked/)).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Initialize & Push' })).toBeDisabled()
    })
})

describe('SettingsView tab routing', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('restores the active tab from the hash', async () => {
        window.history.pushState(null, '', '/settings/doctor')
        stubAPI({ 'GET /api/settings': getSettings, 'GET /api/github/auth': getEmptyGitHub })
        renderSettings()
        expect(await screen.findByRole('tab', { name: 'Doctor' })).toHaveAttribute('aria-selected', 'true')
    })

    it('writes the clicked tab into the hash', async () => {
        window.history.pushState(null, '', '/settings')
        stubAPI({ 'GET /api/settings': getSettings, 'GET /api/github/auth': getEmptyGitHub })
        renderSettings()
        await userEvent.click(await screen.findByRole('tab', { name: 'Git remote' }))
        expect(window.location.pathname).toBe('/settings/git')
    })
})

describe('SettingsView default tab route', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('writes the General default into the URL when arriving at bare #/settings', async () => {
        window.history.pushState(null, '', '/settings')
        stubAPI({ 'GET /api/settings': getSettings, 'GET /api/github/auth': getEmptyGitHub })
        renderSettings()
        expect(await screen.findByRole('tab', { name: 'General' })).toHaveAttribute('aria-selected', 'true')
        await waitFor(() => expect(window.location.pathname).toBe('/settings/general'))
    })
})
