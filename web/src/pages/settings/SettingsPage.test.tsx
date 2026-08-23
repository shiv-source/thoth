import { act, fireEvent, screen, waitFor } from '@testing-library/react'
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
    context_injection: false
}

const githubProvider = {
    id: 1,
    slug: 'github',
    name: 'GitHub',
    driver: 'github',
    kind: 'git' as const,
    base_url: 'https://github.com',
    protected: false,
    fields: [
        { key: 'token', label: 'GitHub token', secret: true, required: true },
        { key: 'owner', label: 'Owner', secret: false, required: false }
    ],
    connection_count: 1
}

const localProvider = {
    id: 2,
    slug: 'local-backup',
    name: 'Local backup',
    driver: 'local',
    kind: 'local' as const,
    base_url: '',
    protected: true,
    fields: [{ key: 'path', label: 'Backup folder', secret: false, required: false }],
    connection_count: 1
}

const githubConnection = {
    id: 3,
    provider_id: 1,
    provider_slug: 'github',
    provider_name: 'GitHub',
    name: 'home',
    enabled: true,
    protected: false,
    active: true,
    identity: { username: 'octo', display_name: 'Octo Cat', email: 'octo@example.com' },
    config: { repo_url: 'https://github.com/octo/wiki.git', has_token: true },
    last_synced_at: '2026-08-23T09:00:00Z',
    last_error: ''
}

const localConnection = {
    id: 4,
    provider_id: 2,
    provider_slug: 'local-backup',
    provider_name: 'Local backup',
    name: 'Local backup',
    enabled: true,
    protected: true,
    active: false,
    identity: null,
    config: { path: '~/.thoth/backups' },
    last_synced_at: '',
    last_error: ''
}

const targets = [
    {
        full_name: 'octo/wiki',
        url: 'https://github.com/octo/wiki.git',
        private: true,
        description: 'My personal knowledge base'
    },
    {
        full_name: 'octo/public-wiki',
        url: 'https://github.com/octo/public-wiki.git',
        private: false,
        description: ''
    }
]

const getSettings = () => settings
// The git connection's targets are fetched lazily by the page.
const getTargets = () => ({ targets })

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
        stubAPI({ 'GET /api/v1/settings': getSettings })

        const { store } = renderSettings()
        // Seed the health slice the way the real API would: the fulfilled
        // action carries the dev payload straight into the reducer. The
        // dispatch re-renders SettingsGeneralPage (it reads default_wiki_path
        // for the hint), so it must run inside act.
        act(() => {
            store.dispatch(fetchHealth.fulfilled(devHealth, 'test', undefined))
        })
        expect(await screen.findByText(/Defaults to ~\/\.thoth\/dev\/wiki/)).toBeInTheDocument()
    })

    it('selects a model from the models endpoint and saves it', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
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
                                provider: 'DeepSeek',
                                provider_id: 1
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

    it('adds a provider with its credentials', async () => {
        const seeded = {
            id: 1,
            name: 'DeepSeek',
            base_url: 'https://api.deepseek.com',
            has_api_key: true,
            model_count: 0
        }
        let providers: (typeof seeded)[] = []
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/models': () => ({ groups: [] }),
            'GET /api/v1/providers': () => ({ providers }),
            'POST /api/v1/providers': () => {
                providers = [seeded]
                return seeded
            }
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        await userEvent.click(await screen.findByRole('button', { name: /Add provider/ }))
        await userEvent.type(await screen.findByLabelText('Name'), 'DeepSeek')
        await userEvent.type(screen.getByLabelText('Base URL'), 'https://api.deepseek.com')
        await userEvent.type(screen.getByLabelText('API key'), 'ds-secret')
        await userEvent.click(screen.getByRole('button', { name: 'OK' }))

        expect(await screen.findByText('Provider added')).toBeInTheDocument()
        expect(await screen.findByText('DeepSeek')).toBeInTheDocument()
        const body = JSON.stringify(lastBody('post', '/api/v1/providers'))
        expect(body).toContain('"name":"DeepSeek"')
        expect(body).toContain('"base_url":"https://api.deepseek.com"')
        expect(body).toContain('"api_key":"ds-secret"')
    })

    it('edits a provider and leaves a blank key untouched', async () => {
        const seeded = {
            id: 1,
            name: 'DeepSeek',
            base_url: 'https://api.deepseek.com',
            has_api_key: true,
            model_count: 0
        }
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/models': () => ({ groups: [] }),
            'GET /api/v1/providers': () => ({ providers: [seeded] }),
            'PUT /api/v1/providers/1': () => ({ ...seeded, base_url: 'https://api.deepseek.com/v1' })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        expect(await screen.findByText('DeepSeek')).toBeInTheDocument()
        await userEvent.click(await screen.findByRole('button', { name: 'Edit DeepSeek' }))
        const baseURL = await screen.findByLabelText('Base URL')
        await userEvent.clear(baseURL)
        await userEvent.type(baseURL, 'https://api.deepseek.com/v1')
        await userEvent.click(screen.getByRole('button', { name: 'OK' }))

        expect(await screen.findByText('Provider updated')).toBeInTheDocument()
        // The api key is write-only: a blank edit sends no key and the base
        // URL round-trips.
        const body = JSON.stringify(lastBody('put', '/api/v1/providers/1'))
        expect(body).toContain('https://api.deepseek.com/v1')
        expect(body).not.toContain('api_key')
    })

    it('shows the save error when the server rejects', async () => {
        stubAPI({ 'GET /api/v1/settings': getSettings })
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
})

describe('SettingsPage wiki export/import', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('exports the wiki as a zip', async () => {
        stubAPI({ 'GET /api/v1/settings': getSettings, 'GET /api/v1/wiki/export': () => new Blob(['zip']) })
        // jsdom has no createObjectURL; the export only needs the call to
        // resolve and build an anchor download.
        vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:mock'), revokeObjectURL: vi.fn() })

        renderSettings()
        await userEvent.click(await screen.findByRole('button', { name: /^Export$/ }))

        expect(await screen.findByText('Wiki exported')).toBeInTheDocument()
        expect(mocks.get.mock.calls.some(([u]) => u === '/api/v1/wiki/export')).toBe(true)
    })

    it('exports with history when requested', async () => {
        stubAPI({ 'GET /api/v1/settings': getSettings, 'GET /api/v1/wiki/export': () => new Blob(['zip']) })
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
            'POST /api/v1/wiki/import': () => ({ files: 7, backup: '/tmp/wiki-backup-20260823-093000' })
        })

        renderSettings()
        const file = new File(['zip'], 'wiki.zip', { type: 'application/zip' })
        await userEvent.upload(document.querySelector('input[type="file"]') as HTMLInputElement, file)

        expect(await screen.findByText('Imported 7 files — a backup was saved')).toBeInTheDocument()
        expect(mocks.post.mock.calls.some(([u]) => u === '/api/v1/wiki/import')).toBe(true)
    })

    it('surfaces an import error from the server', async () => {
        stubAPI({ 'GET /api/v1/settings': getSettings })
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'archive is not a wiki' }))

        renderSettings()
        const file = new File(['zip'], 'wiki.zip', { type: 'application/zip' })
        await userEvent.upload(document.querySelector('input[type="file"]') as HTMLInputElement, file)

        expect(await screen.findByText('archive is not a wiki')).toBeInTheDocument()
    })
})

describe('SettingsPage Providers tab', () => {
    const vendor = { id: 1, name: 'Vendor', base_url: '', has_api_key: false, model_count: 1 }
    const seeded = { id: 1, value: 'my-model', name: 'My Model', tag: 'test', provider: 'Vendor', provider_id: 1 }

    it('lists models and adds one under the provider', async () => {
        // Stateful stubs: mutations refetch the registry, so the GET handler
        // must return the updated list after the POST.
        let list = [seeded]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: list }] }),
            'GET /api/v1/providers': () => ({ providers: [vendor] }),
            'POST /api/v1/models': () => {
                const created = {
                    id: 2,
                    value: 'new-model',
                    name: 'New Model',
                    tag: '',
                    provider: 'Vendor',
                    provider_id: 1
                }
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
        // Adding from a provider panel pre-fills the provider select, so the
        // POST carries that provider's id.
        await userEvent.click(screen.getByRole('button', { name: 'OK' }))

        expect(await screen.findByText('New Model')).toBeInTheDocument()
        const body = JSON.stringify(lastBody('post', '/api/v1/models'))
        expect(body).toContain('new-model')
        expect(body).toContain('"provider_id":1')
    })

    it('edits a model', async () => {
        let list = [seeded]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: list }] }),
            'GET /api/v1/providers': () => ({ providers: [vendor] }),
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
            'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: list }] }),
            'GET /api/v1/providers': () => ({ providers: [vendor] }),
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
            'GET /api/v1/models': () => ({ groups: [] }),
            'GET /api/v1/providers': () => ({ providers: [] })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        expect(await screen.findByText('No providers yet')).toBeInTheDocument()
        // The empty-state CTA opens the add-provider modal.
        await userEvent.click(await screen.findByText('Add your first provider'))
        expect(await screen.findByLabelText('Name')).toBeInTheDocument()
    })

    it('deletes a provider and removes its models', async () => {
        let providers = [{ id: 7, name: 'Doomed', base_url: '', has_api_key: false, model_count: 1 }]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/models': () => ({ groups: [] }),
            'GET /api/v1/providers': () => ({ providers }),
            'DELETE /api/v1/providers/7': () => {
                providers = []
                return { ok: true }
            }
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Providers' }))
        expect(await screen.findByText('Doomed')).toBeInTheDocument()
        await userEvent.click(await screen.findByRole('button', { name: 'Delete Doomed' }))
        await userEvent.click(await screen.findByRole('button', { name: 'OK' }))

        expect(await screen.findByText('Provider deleted')).toBeInTheDocument()
        expect(mocks.delete.mock.calls.some(([u]) => u === '/api/v1/providers/7')).toBe(true)
    })
})

describe('SettingsPage Sync tab', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('lists connected destinations with their identity and last sync', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [githubConnection, localConnection] }),
            'GET /api/v1/sync/connections/3/targets': getTargets
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        // The GitHub connection shows its identity and the chosen repository;
        // the local backup is tagged protected.
        expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
        expect(await screen.findByText('octo/wiki')).toBeInTheDocument()
        expect(screen.getByText('Last synced 2026-08-23')).toBeInTheDocument()
        expect(screen.getByText('Local backup')).toBeInTheDocument()
        expect(screen.getAllByText('Protected').length).toBeGreaterThan(0)
    })

    it('connects a GitHub destination through the provider-driven form', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [] }),
            'POST /api/v1/sync/connections': () => githubConnection,
            'GET /api/v1/sync/connections/3/targets': getTargets
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        expect(await screen.findByText('No destinations connected yet — connect one below')).toBeInTheDocument()

        // The provider select drives the credential fields.
        await userEvent.click(screen.getByRole('combobox', { name: 'Provider' }))
        await userEvent.click(await screen.findByRole('option', { name: 'GitHub' }))
        await userEvent.type(await screen.findByLabelText('Name'), 'home')
        await userEvent.type(screen.getByLabelText('GitHub token'), 'ghp_secret')
        await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

        // The connection card appears with the identity the server verified.
        expect(await screen.findByText('Octo Cat')).toBeInTheDocument()
        expect(await screen.findByText('Destination connected')).toBeInTheDocument()
        const body = JSON.stringify(lastBody('post', '/api/v1/sync/connections'))
        expect(body).toContain('"provider_id":1')
        expect(body).toContain('"name":"home"')
        expect(body).toContain('ghp_secret')
    })

    it('shows the connect error from the server', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [] })
        })
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'github rejected the token' }))

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        await userEvent.click(await screen.findByRole('combobox', { name: 'Provider' }))
        await userEvent.click(await screen.findByRole('option', { name: 'GitHub' }))
        await userEvent.type(await screen.findByLabelText('Name'), 'home')
        await userEvent.type(screen.getByLabelText('GitHub token'), 'bad-token')
        await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

        // The message renders inline and as a toast.
        expect((await screen.findAllByText('github rejected the token')).length).toBeGreaterThan(0)
    })

    it('pushes a connection to its destination', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [githubConnection] }),
            'GET /api/v1/sync/connections/3/targets': getTargets,
            'POST /api/v1/sync/connections/3/push': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        await userEvent.click(await screen.findByRole('button', { name: 'Push now' }))

        expect(await screen.findByText('Wiki pushed')).toBeInTheDocument()
        expect(mocks.post.mock.calls.some(([u]) => u === '/api/v1/sync/connections/3/push')).toBe(true)
    })

    it('surfaces a push failure from the destination', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [githubConnection] }),
            'GET /api/v1/sync/connections/3/targets': getTargets,
            'POST /api/v1/sync/connections/3/push': () => ({ ok: false, error: 'remote rejected the branch' })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        await userEvent.click(await screen.findByRole('button', { name: 'Push now' }))

        // The destination error surfaces as a toast and in the page alert.
        expect((await screen.findAllByText('remote rejected the branch')).length).toBeGreaterThan(0)
    })

    it('disconnects a destination and drops its card', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [githubConnection, localConnection] }),
            'GET /api/v1/sync/connections/3/targets': getTargets,
            'DELETE /api/v1/sync/connections/3': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        // Only the unprotected GitHub card offers Disconnect.
        expect(await screen.findByRole('button', { name: 'Disconnect' })).toBeInTheDocument()
        await userEvent.click(screen.getByRole('button', { name: 'Disconnect' }))

        expect(await screen.findByText('Disconnected')).toBeInTheDocument()
        await waitFor(() => expect(screen.queryByText('Octo Cat')).not.toBeInTheDocument())
        expect(mocks.delete.mock.calls.some(([u]) => u === '/api/v1/sync/connections/3')).toBe(true)
    })

    it('sets the active connection', async () => {
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [githubConnection, localConnection] }),
            'GET /api/v1/sync/connections/3/targets': getTargets,
            'POST /api/v1/sync/connections/4/active': () => ({ ok: true })
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        // The active GitHub connection has no Set-active button; the local
        // backup does. Promoting it moves active to the local connection.
        expect(await screen.findByRole('button', { name: 'Set active' })).toBeInTheDocument()
        await userEvent.click(screen.getByRole('button', { name: 'Set active' }))

        await waitFor(() => expect(mocks.post).toHaveBeenCalledWith('/api/v1/sync/connections/4/active'))
    })

    it('adds a custom sync provider and deletes a disposable one', async () => {
        const enterprise = {
            id: 9,
            slug: 'custom-1',
            name: 'GitHub Enterprise',
            driver: 'github',
            kind: 'git' as const,
            base_url: 'https://ghe.example.com',
            protected: false,
            fields: githubProvider.fields,
            connection_count: 0
        }
        let providers = [githubProvider, localProvider]
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers }),
            'GET /api/v1/sync/connections': () => ({ connections: [] }),
            'POST /api/v1/sync/providers': () => {
                providers = [...providers, enterprise]
                return enterprise
            },
            'DELETE /api/v1/sync/providers/9': () => {
                providers = providers.filter((p) => p.id !== 9)
                return { ok: true }
            }
        })

        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        await userEvent.click(
            await screen.findByText('Manage the provider catalog (built-ins + your custom providers)')
        )

        // The local backup is locked; add a new provider via the inline form.
        const deleteLocal = screen.getByRole('button', { name: 'Delete Local backup' })
        expect(deleteLocal).toBeDisabled()
        await userEvent.type(await screen.findByLabelText('Provider name'), 'GitHub Enterprise')
        await userEvent.click(screen.getByRole('combobox', { name: 'Driver' }))
        // antd's dropdown leave motion never completes under jsdom, so the
        // closed popup stays mounted over the inline form and swallows
        // hit-tested clicks. fireEvent dispatches directly, so the option's
        // onClick and the form submit both run.
        fireEvent.click(await screen.findByRole('option', { name: 'GitHub' }))
        await userEvent.type(screen.getByLabelText('Base URL'), 'https://ghe.example.com')
        fireEvent.click(screen.getByRole('button', { name: 'Add provider' }))

        expect(await screen.findByText('GitHub Enterprise')).toBeInTheDocument()

        expect(await screen.findByText('GitHub Enterprise')).toBeInTheDocument()

        expect(await screen.findByText('GitHub Enterprise')).toBeInTheDocument()
        const addBody = JSON.stringify(lastBody('post', '/api/v1/sync/providers'))
        expect(addBody).toContain('"name":"GitHub Enterprise"')
        expect(addBody).toContain('"driver":"github"')

        // The custom provider is disposable.
        await userEvent.click(screen.getByRole('button', { name: 'Delete GitHub Enterprise' }))
        await waitFor(() => expect(screen.queryByText('GitHub Enterprise')).not.toBeInTheDocument())
        expect(mocks.delete.mock.calls.some(([u]) => u === '/api/v1/sync/providers/9')).toBe(true)
    })
})

describe('SettingsPage tab routing', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('restores the active tab from the hash', async () => {
        window.history.pushState(null, '', '/settings/doctor')
        stubAPI({ 'GET /api/v1/settings': getSettings })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'Doctor' })).toHaveClass('ant-menu-item-selected')
    })

    it('restores the Providers tab from the hash', async () => {
        window.history.pushState(null, '', '/settings/providers')
        stubAPI({ 'GET /api/v1/settings': getSettings })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'Providers' })).toHaveClass('ant-menu-item-selected')
    })

    it('restores the Sync tab from the hash', async () => {
        window.history.pushState(null, '', '/settings/sync')
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [] })
        })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'Sync' })).toHaveClass('ant-menu-item-selected')
    })

    it('writes the clicked tab into the hash', async () => {
        window.history.pushState(null, '', '/settings')
        stubAPI({
            'GET /api/v1/settings': getSettings,
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [] })
        })
        renderSettings()
        await userEvent.click(await screen.findByRole('menuitem', { name: 'Sync' }))
        expect(window.location.pathname).toBe('/settings/sync')
    })
})

describe('SettingsPage default tab route', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('writes the General default into the URL when arriving at bare #/settings', async () => {
        window.history.pushState(null, '', '/settings')
        stubAPI({ 'GET /api/v1/settings': getSettings })
        renderSettings()
        expect(await screen.findByRole('menuitem', { name: 'General' })).toHaveClass('ant-menu-item-selected')
        await waitFor(() => expect(window.location.pathname).toBe('/settings/general'))
    })
})
