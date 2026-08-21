import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchHealth } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import type { Health } from '../api/client'
import { AppSider } from './AppSider'

const mocks = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('axios', () => ({
    default: {
        create: () => ({ get: mocks.get }),
        isAxiosError: (e: unknown) =>
            !!(e && typeof e === 'object' && (e as { isAxiosError?: boolean }).isAxiosError === true)
    }
}))

const healthy: Health = {
    status: 'ok',
    backend: { name: 'thoth-agent', api_key_configured: true, model: 'claude-sonnet-5', provider: 'Anthropic' },
    wiki: { path: '/tmp/wiki', exists: true },
    version: '1.2.3',
    dev: false,
    commit: '',
    default_wiki_path: '~/.thoth/wiki'
}

function stubHealth(health: Health) {
    mocks.get.mockImplementation(() => Promise.resolve({ data: health }))
}

describe('AppSider navigation', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('renders every menu item and selects the active view', () => {
        window.history.pushState(null, '', '/notes')
        renderWithStore(<AppSider />)
        expect(screen.getByRole('menuitem', { name: 'Chat' })).toBeInTheDocument()
        // antd v6 marks the selected item with a class, not an aria
        // attribute — the class is the component's own contract here.
        expect(screen.getByRole('menuitem', { name: 'Notes' })).toHaveClass('ant-menu-item-selected')
        expect(screen.getByRole('menuitem', { name: 'Dashboard' })).not.toHaveClass('ant-menu-item-selected')
        expect(screen.getByRole('menuitem', { name: 'Search' })).toBeInTheDocument()
        expect(screen.getByRole('menuitem', { name: 'Settings' })).toBeInTheDocument()
    })

    it('navigates views through the path on click', async () => {
        renderWithStore(<AppSider />)
        await userEvent.click(screen.getByRole('menuitem', { name: 'Dashboard' }))
        expect(window.location.pathname).toBe('/')
        expect(screen.getByRole('menuitem', { name: 'Dashboard' })).toHaveClass('ant-menu-item-selected')
    })
})

describe('AppSider health footer', () => {
    it('shows the healthy state with the version', async () => {
        stubHealth(healthy)
        const { store } = renderWithStore(<AppSider />)
        void store.dispatch(fetchHealth())
        expect(await screen.findByText('All systems go')).toBeInTheDocument()
        expect(screen.getByText('v1.2.3')).toBeInTheDocument()
    })

    it('shows the missing-key state', async () => {
        stubHealth({ ...healthy, backend: { ...healthy.backend, api_key_configured: false } })
        const { store } = renderWithStore(<AppSider />)
        void store.dispatch(fetchHealth())
        expect(await screen.findByText('API key not configured')).toBeInTheDocument()
    })

    it('shows the loading state until health resolves', async () => {
        mocks.get.mockImplementation(() => new Promise(() => {}))
        renderWithStore(<AppSider />)
        expect(screen.getByText('Checking…')).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled())
    })
})
