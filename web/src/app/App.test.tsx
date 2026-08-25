import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { fetchHealth } from '../store'
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
        isAxiosError: () => false
    }
}))

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

const notConfigured = {
    status: 'ok',
    backend: { name: 'thoth-agent', api_key_configured: false, model: '', provider: '' },
    wiki: { path: '~/.thoth/wiki', exists: true },
    version: '1.2.3',
    dev: false,
    commit: '',
    default_wiki_path: '~/.thoth/wiki'
}

const settings = {
    wiki_path: '~/.thoth/wiki',
    wiki_folders: [] as string[],
    model: '',
    context_injection: false,
    conversation_retention_days: 7
}

describe('App setup gating', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('lets the user open Settings while no API key is configured', async () => {
        stubAPI({
            'GET /api/v1/health': () => notConfigured,
            'GET /api/v1/settings': () => settings,
            'GET /api/v1/models': () => ({ groups: [] })
        })

        // main.tsx dispatches fetchHealth on boot; the test replicates that.
        const { store } = renderWithStore(<App />)
        void store.dispatch(fetchHealth())
        // Setup gates the content area and names Settings as the fix.
        expect(await screen.findByText('Thoth needs your attention')).toBeInTheDocument()
        expect(await screen.findByText(/Add your API key in Settings/)).toBeInTheDocument()
        // The Settings page is reachable anyway — the key must be entered
        // somewhere to unblock setup.
        await userEvent.click(await screen.findByRole('menuitem', { name: /Settings/ }))
        expect(await screen.findByRole('menuitem', { name: 'General' })).toHaveClass('ant-menu-item-selected')
    })
})
