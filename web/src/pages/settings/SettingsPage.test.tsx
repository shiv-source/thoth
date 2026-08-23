import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SettingsPage } from './SettingsPage'
import { fetchHealth } from '../../store'
import { renderWithStore } from '../../test/renderWithStore'

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
    wiki_folders: [] as string[],
    model: '',
    providers: {} as Record<string, { api_key?: string; has_api_key: boolean; base_url: string }>,
    repo_url: '',
    sync_enabled: false,
    context_injection: false
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
    return renderWithStore(<SettingsPage />)
}

// The dev banner's data: health carries dev + commit, and the server-side
// wiki default (the settings hint reads it from the API, not from a
// hardcoded string).
const devHealth = {
    status: 'ok',
    backend: { name: 'thoth-agent', api_key_configured: true, model: 'claude-sonnet-5', provider: 'Anthropic' },
    wiki: { path: '/tmp/wiki', exists: true },
    version: 'dev',
    dev: true,
    commit: '3ec01b868d6b5aa2e87505e8f56f5928fbbcce1c',
    default_wiki_path: '~/.thoth/dev/wiki'
}

describe('SettingsPage', () => {
    // Tab clicks ride the URL; reset it so a test never inherits the
    // previous test's active tab.
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('loads current settings and saves edits', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'PUT /api/v1/settings': () => ({ ...settings, wiki_path: '/tmp/other/wiki' })
        })

        renderSettings()
        const wikiPath = await screen.findByDisplayValue('~/.thoth/wiki')
        // The prod hint names the production default.
        expect(screen.getByText(/Defaults to ~\/\.thoth\/wiki/)).toBeInTheDocument()
        await userEvent.clear(wikiPath)
        await userEvent.type(wikiPath, '/tmp/other/wiki')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
        // The save also surfaces as a toast.
        expect(await screen.findByText('Settings saved')).toBeInTheDocument()
    })

    it('saves a custom scaffold folder set from the General tab', async () => {
        stubAPI({
            'GET /api/v1/settings': () => ({ ...settings, wiki_folders: ['inbox', 'meetings'] }),
            'GET /api/v1/github/auth': getEmptyGitHub,
            'PUT /api/v1/settings': () => ({ ...settings, wiki_folders: ['inbox', 'meetings', 'journal'] })
        })

        renderSettings()
        // The configured set renders as tags.
        expect(await screen.findByText('inbox')).toBeInTheDocument()
        expect(await screen.findByText('meetings')).toBeInTheDocument()
        // Typing a new name adds it to the set.
        const folderInput = await screen.findByRole('combobox', { name: 'Scaffold folders' })
        await userEvent.type(folderInput, 'journal{Enter}')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
        const put = lastBody('put', '/api/v1/settings')
        expect(put != null && JSON.stringify(put).includes('"journal"')).toBe(true)
    })

    it('names the dev wiki default in the hint when the server runs in dev mode', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub
        })

        const { store } = renderSettings()
        // Seed the health slice the way the real API would: the fulfilled
        // action carries the dev payload straight into the reducer.
        store.dispatch(fetchHealth.fulfilled(devHealth, 'test', undefined))
        expect(await screen.findByText(/Defaults to ~\/\.thoth\/dev\/wiki/)).toBeInTheDocument()
    })

    it('selects a model from the models endpoint and saves it', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({
                groups: [
                    {
                        provider: 'DeepSeek',
                        models: [
                            {
                                id: 1,
                                value: 'deepseek-v4-flash',
                                name: 'V4 Flash',
                                tag: 'fastest',
                                provider: 'DeepSeek'
                            }
                        ]
                    }
                ]
            }),
            'PUT /api/v1/settings': () => ({ ...settings })
        })

        renderSettings()
        // The Model select only lists the chosen provider's models, so pick
        // the provider first.
        await userEvent.click(await screen.findByRole('combobox', { name: 'Provider' }))
        await userEvent.click(await screen.findByRole('option', { name: 'DeepSeek' }))
        await userEvent.click(await screen.findByRole('combobox', { name: 'Model' }))
        // The option renders name + tag as secondary text.
        await userEvent.click(await screen.findByRole('option', { name: /V4 Flash/ }))
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        await waitFor(() => expect(screen.getByText(/Saved ✓/)).toBeInTheDocument())
        expect(lastBody('put', '/api/v1/settings')).toMatchObject({ model: 'deepseek-v4-flash' })
    })

    it('renders and saves per-provider credentials', async () => {
        stubAPI({
            'GET /api/v1/settings': () => ({
                ...settings,
                providers: { DeepSeek: { api_key: '', has_api_key: true, base_url: 'https://api.deepseek.com' } }
            }),
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({
                groups: [
                    {
                        provider: 'DeepSeek',
                        models: [
                            {
                                id: 1,
                                value: 'deepseek-v4-flash',
                                name: 'V4 Flash',
                                tag: 'fastest',
                                provider: 'DeepSeek'
                            }
                        ]
                    }
                ]
            }),
            'PUT /api/v1/settings': () => ({ ...settings })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        // The only provider's panel is open by default with the saved base URL.
        expect(await screen.findByText('DeepSeek')).toBeInTheDocument()
        const baseURL = await screen.findByLabelText('Base URL')
        expect(baseURL).toHaveValue('https://api.deepseek.com')
        await userEvent.clear(baseURL)
        await userEvent.type(baseURL, 'https://api.deepseek.com/v1')
        // The per-provider key input is the '•••••••• (saved)' one (the
        // fallback key is unset, so its placeholder is sk-ant-…).
        await userEvent.type(await screen.findByPlaceholderText('•••••••• (saved)'), 'ds-new-key')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        expect(await screen.findByText('Settings saved')).toBeInTheDocument()

        const body = JSON.stringify(lastBody('put', '/api/v1/settings'))
        expect(body).toContain('https://api.deepseek.com/v1')
        expect(body).toContain('ds-new-key')
    })

    it('saves a provider base URL without touching an existing key', async () => {
        stubAPI({
            'GET /api/v1/settings': () => ({
                ...settings,
                providers: { OpenAI: { api_key: '', has_api_key: true, base_url: '' } }
            }),
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({
                groups: [
                    {
                        provider: 'OpenAI',
                        models: [{ id: 2, value: 'gpt-5.6-mini', name: 'Mini', tag: 'fast', provider: 'OpenAI' }]
                    }
                ]
            }),
            'PUT /api/v1/settings': () => ({ ...settings })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        const baseURL = await screen.findByLabelText('Base URL')
        await userEvent.type(baseURL, 'https://api.openai.com/v1')
        await userEvent.click(screen.getByRole('button', { name: /Save/ }))
        expect(await screen.findByText('Settings saved')).toBeInTheDocument()

        // The per-provider api key stays empty in the PUT body (write-only
        // leave-unchanged semantics are the server's), the base URL goes.
        const put = lastBody('put', '/api/v1/settings') as { providers?: Record<string, unknown> }
        expect(put.providers?.['OpenAI']).toMatchObject({ base_url: 'https://api.openai.com/v1' })
    })

    it('shows the save error when the server rejects', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub
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
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/doctor': () => ({
                checks: [
                    { name: 'wiki', ok: true, message: '/tmp/wiki exists with the 8 scaffold folders and CLAUDE.md' },
                    {
                        name: 'provider',
                        ok: false,
                        message: 'Anthropic rejected the API key (401) — set a valid one in Settings → Providers'
                    }
                ]
            })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Doctor' }))
        expect(await screen.findByText('wiki')).toBeInTheDocument()
        expect(
            screen.getByText('Anthropic rejected the API key (401) — set a valid one in Settings → Providers')
        ).toBeInTheDocument()
    })

    it('stores the git remote URL and pushes', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': () => connected,
            'GET /api/v1/github/repos': getRepos,
            'POST /api/v1/git/setup': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        // The connected account's repos are offered as a dropdown the width of
        // the input; picking the private repo fills the field.
        const url = await screen.findByLabelText('Git remote URL')
        await userEvent.type(url, 'octo')
        const option = await screen.findByRole('option', { name: /octo\/wiki/ })
        expect(await screen.findByText('My personal knowledge base')).toBeInTheDocument()
        await userEvent.click(option)
        expect(url).toHaveValue('https://github.com/octo/wiki.git')
        await userEvent.click(screen.getByRole('button', { name: 'Initialize & Push' }))
        expect(await screen.findByText('Wiki pushed to remote')).toBeInTheDocument()
        // The setup call carried the URL.
        const gitBody = lastBody('post', '/api/v1/git/setup')
        expect(gitBody != null && JSON.stringify(gitBody).includes('https://github.com/octo/wiki.git')).toBe(true)
    })

    it('connects a GitHub account with a token', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/github/repos': getRepos,
            'POST /api/v1/github/auth': () => connected
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        await userEvent.type(await screen.findByPlaceholderText(/ghp_/), 'ghp_secret123')
        await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

        expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
        expect(screen.getByText('GitHub connected')).toBeInTheDocument()
        // The stored account facts render view-only.
        expect(screen.getByText(/Member since 2018-05-01/)).toBeInTheDocument()
        // The remote URL input appears once connected.
        expect(await screen.findByLabelText('Git remote URL')).toBeInTheDocument()
        // The POST carried the token.
        const connectBody = lastBody('post', '/api/v1/github/auth')
        expect(connectBody != null && JSON.stringify(connectBody).includes('ghp_secret123')).toBe(true)
    })

    it('shows the connect error', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub
        })
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'github rejected the token' }))

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        await userEvent.type(await screen.findByPlaceholderText(/ghp_/), 'bad-token')
        await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

        // The message renders inline and as a toast.
        expect((await screen.findAllByText('github rejected the token')).length).toBeGreaterThan(0)
    })

    it('disconnects a GitHub account', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': () => connected,
            'GET /api/v1/github/repos': getRepos,
            'DELETE /api/v1/github/auth': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
        await userEvent.click(screen.getByRole('button', { name: 'Disconnect' }))

        expect(await screen.findByPlaceholderText(/ghp_/)).toBeInTheDocument()
        expect(await screen.findByText('GitHub disconnected')).toBeInTheDocument()
        expect(mocks.delete.mock.calls.some(([u]) => u === '/api/v1/github/auth')).toBe(true)
    })
})

describe('SettingsPage wiki export/import', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('exports the wiki as a zip', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/wiki/export': () => new Blob(['zip'])
        })
        // jsdom has no createObjectURL; the export only needs the call to
        // resolve and build an anchor download.
        vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:mock'), revokeObjectURL: vi.fn() })

        renderSettings()
        await userEvent.click(await screen.findByRole('button', { name: /^Export$/ }))

        expect(await screen.findByText('Wiki exported')).toBeInTheDocument()
        expect(mocks.get.mock.calls.some(([u]) => u === '/api/v1/wiki/export')).toBe(true)
    })

    it('exports with history when requested', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/wiki/export': () => new Blob(['zip'])
        })
        vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:mock'), revokeObjectURL: vi.fn() })

        renderSettings()
        await userEvent.click(await screen.findByRole('button', { name: /Export with history/ }))

        expect(await screen.findByText('Wiki exported')).toBeInTheDocument()
        const call = [...mocks.get.mock.calls].reverse().find(([u]) => u === '/api/v1/wiki/export')
        const config = call?.[1] as { params?: { history?: string } } | undefined
        expect(config?.params?.history).toBe('1')
    })

    it('imports a wiki zip and reports the merged count', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'POST /api/v1/wiki/import': () => ({ files: 7, backup: '/tmp/wiki-backup-20260823-093000' })
        })

        renderSettings()
        const file = new File(['zip'], 'wiki.zip', { type: 'application/zip' })
        await userEvent.upload(document.querySelector('input[type="file"]') as HTMLInputElement, file)

        expect(await screen.findByText('Imported 7 files — a backup was saved')).toBeInTheDocument()
        expect(mocks.post.mock.calls.some(([u]) => u === '/api/v1/wiki/import')).toBe(true)
    })

    it('surfaces an import error from the server', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub
        })
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'archive is not a wiki' }))

        renderSettings()
        const file = new File(['zip'], 'wiki.zip', { type: 'application/zip' })
        await userEvent.upload(document.querySelector('input[type="file"]') as HTMLInputElement, file)

        expect(await screen.findByText('archive is not a wiki')).toBeInTheDocument()
    })
})

describe('SettingsPage Providers tab', () => {
    const seeded = { id: 1, value: 'my-model', name: 'My Model', tag: 'test', provider: 'Vendor' }

    it('lists models and adds one', async () => {
        // Stateful stubs: mutations refetch the registry, so the GET handler
        // must return the updated list after the POST.
        let list = [seeded]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: list }] }),
            'POST /api/v1/models': () => {
                const created = { id: 2, value: 'new-model', name: 'New Model', tag: '', provider: 'Vendor' }
                list = [...list, created]
                return created
            }
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        // The single provider's panel is open by default; its header names
        // it and counts its models, and the table lists the model with its
        // tag rendered as a chip.
        expect(await screen.findByText('My Model')).toBeInTheDocument()
        expect(screen.getByText('Vendor')).toBeInTheDocument()
        expect(screen.getByText('1 model')).toBeInTheDocument()
        expect(screen.getByText('test')).toBeInTheDocument()

        await userEvent.click(screen.getByRole('button', { name: 'Add model' }))
        await userEvent.type(await screen.findByLabelText('Value'), 'new-model')
        await userEvent.type(screen.getByLabelText('Name'), 'New Model')
        // Adding from a provider panel pre-fills the provider field (the
        // placeholder scopes the query — the hidden General tab has its own
        // "Provider" label).
        expect(screen.getByPlaceholderText('Anthropic')).toHaveValue('Vendor')
        await userEvent.click(screen.getByRole('button', { name: 'OK' }))

        expect(await screen.findByText('New Model')).toBeInTheDocument()
        expect(JSON.stringify(lastBody('post', '/api/v1/models'))).toContain('new-model')
    })

    it('edits a model', async () => {
        let list = [seeded]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: list }] }),
            'PUT /api/v1/models/1': () => {
                list = [{ ...seeded, name: 'Renamed' }]
                return list[0]!
            }
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        // byText, not byRole: role-name computation hangs under jsdom for
        // buttons inside the antd Table.
        await userEvent.click(await screen.findByText('Edit'))
        const name = await screen.findByLabelText('Name')
        await userEvent.clear(name)
        await userEvent.type(name, 'Renamed')
        await userEvent.click(screen.getByRole('button', { name: 'OK' }))

        expect(await screen.findByText('Renamed')).toBeInTheDocument()
        expect(JSON.stringify(lastBody('put', '/api/v1/models/1'))).toContain('Renamed')
    })

    it('deletes a model', async () => {
        let list = [seeded]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: list }] }),
            'DELETE /api/v1/models/1': () => {
                list = []
                return { ok: true }
            }
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        expect(await screen.findByText('My Model')).toBeInTheDocument()
        await userEvent.click(await screen.findByText('Delete'))
        await userEvent.click(await screen.findByText('OK'))

        await waitFor(() => expect(screen.queryByText('My Model')).not.toBeInTheDocument())
        expect(mocks.delete.mock.calls.some(([u]) => u === '/api/v1/models/1')).toBe(true)
    })

    it('shows an empty state with a call to action', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': getEmptyGitHub,
            'GET /api/v1/models': () => ({ groups: [] })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        expect(await screen.findByText('No models yet')).toBeInTheDocument()
        // The empty-state CTA opens the same add modal.
        await userEvent.click(await screen.findByText('Add your first model'))
        expect(await screen.findByLabelText('Value')).toBeInTheDocument()
    })
})

describe('SettingsPage auto-sync', () => {
    it('toggles sync_enabled in the Git tab and saves it', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': () => connected,
            'GET /api/v1/github/repos': getRepos,
            'PUT /api/v1/settings': () => ({ ...settings, sync_enabled: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        await userEvent.click(await screen.findByRole('switch'))
        await userEvent.click(screen.getByRole('button', { name: 'Save' }))

        expect(await screen.findByText('Settings saved')).toBeInTheDocument()
        const put = lastBody('put', '/api/v1/settings')
        expect(put != null && JSON.stringify(put).includes('"sync_enabled":true')).toBe(true)
    })
})

describe('SettingsPage public repo guard', () => {
    it('warns and blocks push when a public repo is picked', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/github/auth': () => connected,
            'GET /api/v1/github/repos': getRepos
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        const url = await screen.findByLabelText('Git remote URL')
        await userEvent.type(url, 'octo/public')
        await userEvent.click(await screen.findByRole('option', { name: /octo\/public-wiki/ }))

        expect(url).toHaveValue('https://github.com/octo/public-wiki.git')
        expect(await screen.findByText(/public repository is blocked/)).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Initialize & Push' })).toBeDisabled()
    })
})

describe('SettingsPage tab routing', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('restores the active tab from the hash', async () => {
        window.history.pushState(null, '', '/settings/doctor')
        stubAPI({ 'GET /api/v1/settings': getSettings, 'GET /api/v1/github/auth': getEmptyGitHub })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'Doctor' })).toHaveClass('ant-menu-item-selected')
    })

    it('restores the Providers tab from the hash', async () => {
        window.history.pushState(null, '', '/settings/providers')
        stubAPI({ 'GET /api/v1/settings': getSettings, 'GET /api/v1/github/auth': getEmptyGitHub })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'Providers' })).toHaveClass('ant-menu-item-selected')
    })

    it('writes the clicked tab into the hash', async () => {
        window.history.pushState(null, '', '/settings')
        stubAPI({ 'GET /api/v1/settings': getSettings, 'GET /api/v1/github/auth': getEmptyGitHub })
        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Git remote' }))
        expect(window.location.pathname).toBe('/settings/git')
    })
})

describe('SettingsPage default tab route', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('writes the General default into the URL when arriving at bare #/settings', async () => {
        window.history.pushState(null, '', '/settings')
        stubAPI({ 'GET /api/v1/settings': getSettings, 'GET /api/v1/github/auth': getEmptyGitHub })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'General' })).toHaveClass('ant-menu-item-selected')
        await waitFor(() => expect(window.location.pathname).toBe('/settings/general'))
    })
})
