import { screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { Health } from '../api/client'
import { renderWithStore } from '../test/renderWithStore'
import { SetupScreen } from './SetupScreen'

const healthy: Health = {
    status: 'ok',
    backend: { name: 'thoth-agent', api_key_configured: true, model: 'claude-sonnet-5', provider: 'Anthropic' },
    wiki: { path: '~/.thoth/wiki', exists: true },
    version: '1.2.3',
    dev: false,
    commit: '',
    default_wiki_path: '~/.thoth/wiki'
}

function render(health: Health | null) {
    return renderWithStore(<SetupScreen health={health} loading={false} onRecheck={() => {}} />)
}

describe('SetupScreen', () => {
    it('shows the native onboarding path when no API key is configured', () => {
        render({ ...healthy, backend: { ...healthy.backend, api_key_configured: false } })
        expect(screen.getByText('No provider API key is configured.')).toBeInTheDocument()
        expect(screen.getByText(/Add your API key in Settings/)).toBeInTheDocument()
    })

    it('renders no problems when the agent is ready', () => {
        render(healthy)
        expect(screen.getByText('Thoth needs your attention')).toBeInTheDocument()
        expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    })

    it('lists the wiki problem when the wiki is missing', () => {
        render({ ...healthy, wiki: { path: '~/.thoth/wiki', exists: false } })
        expect(screen.getByText('Your wiki directory does not exist yet.')).toBeInTheDocument()
        expect(screen.getByText('thoth init')).toBeInTheDocument()
    })

    it('reports an unreachable server when health is null', () => {
        render(null)
        expect(screen.getByText('The Thoth server is unreachable.')).toBeInTheDocument()
    })

    it('re-checks on button click', async () => {
        const onRecheck = vi.fn()
        renderWithStore(<SetupScreen health={null} loading={false} onRecheck={onRecheck} />)
        const button = await screen.findByRole('button', { name: /Re-check/ })
        button.click()
        expect(onRecheck).toHaveBeenCalledTimes(1)
    })
})
