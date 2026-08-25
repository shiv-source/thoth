import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SaveAsNote } from './SaveAsNote'
import { renderWithStore } from '../../test/renderWithStore'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

describe('SaveAsNote', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        stubAPI(mocks, {
            'GET /api/v1/settings': () => ({
                wiki_path: '~/.thoth/wiki',
                wiki_folders: [],
                model: '',
                providers: {},
                repo_url: '',
                sync_enabled: false,
                context_injection: false,
                conversation_retention_days: 7
            })
        })
    })
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it('opens with the default folder and promotes content on save', async () => {
        stubAPI(mocks, {
            'GET /api/v1/settings': () => ({
                wiki_path: '~/.thoth/wiki',
                wiki_folders: [],
                model: '',
                providers: {},
                repo_url: '',
                sync_enabled: false,
                context_injection: false,
                conversation_retention_days: 7
            }),
            'POST /api/v1/notes': () => ({ path: 'inbox/the-answer.md', title: 'The Answer', type: 'inbox' }),
            'GET /api/v1/wiki/tree': () => ({ nodes: [] })
        })
        const { store } = renderWithStore(<SaveAsNote content="# The Answer" />)

        fireEvent.click(screen.getByRole('button', { name: 'Save as note' }))
        await waitFor(() => expect(screen.getByText('Save as note')).toBeInTheDocument())
        // Default folder is inbox (first scaffold folder).
        expect(screen.getByText('inbox')).toBeInTheDocument()

        fireEvent.click(screen.getByRole('button', { name: 'Save' }))
        await waitFor(() =>
            expect(mocks.post).toHaveBeenCalledWith('/api/v1/notes', { content: '# The Answer', folder: 'inbox' })
        )
        // The save dispatches a "Note saved" notification carrying the path.
        await waitFor(() =>
            expect(store.getState().notifications.items.some((n) => n.body === 'inbox/the-answer.md')).toBe(true)
        )
    })

    it('sends the chosen folder and toasts the saved path', async () => {
        stubAPI(mocks, {
            'GET /api/v1/settings': () => ({
                wiki_path: '~/.thoth/wiki',
                wiki_folders: ['journal', 'recipes'],
                model: '',
                providers: {},
                repo_url: '',
                sync_enabled: false,
                context_injection: false,
                conversation_retention_days: 7
            }),
            'POST /api/v1/notes': () => ({ path: 'journal/j.md', title: 'J', type: 'journal' }),
            'GET /api/v1/wiki/tree': () => ({ nodes: [] })
        })
        const { store } = renderWithStore(<SaveAsNote content="# J" />)

        fireEvent.click(screen.getByRole('button', { name: 'Save as note' }))
        // With configured folders, the picker lists them and defaults to the
        // first once the async settings fetch resolves.
        await waitFor(() => expect(screen.getByText('journal')).toBeInTheDocument())

        await userEvent.click(await screen.findByRole('combobox', { name: 'Target folder' }))
        await userEvent.click(await screen.findByRole('option', { name: 'recipes' }))
        fireEvent.click(screen.getByRole('button', { name: 'Save' }))
        await waitFor(() =>
            expect(mocks.post).toHaveBeenCalledWith('/api/v1/notes', { content: '# J', folder: 'recipes' })
        )
        // The toast carries the saved path.
        await waitFor(() =>
            expect(store.getState().notifications.items.some((n) => n.body === 'journal/j.md')).toBe(true)
        )
    })

    it('shows an error toast when the save fails', async () => {
        stubAPI(mocks, {
            'GET /api/v1/settings': () => ({
                wiki_path: '~/.thoth/wiki',
                wiki_folders: [],
                model: '',
                providers: {},
                repo_url: '',
                sync_enabled: false,
                context_injection: false,
                conversation_retention_days: 7
            })
        })
        mocks.post.mockRejectedValue(new Error('boom'))
        renderWithStore(<SaveAsNote content="# X" />)

        fireEvent.click(screen.getByRole('button', { name: 'Save as note' }))
        await waitFor(() => expect(screen.getByRole('combobox')).toBeInTheDocument())
        fireEvent.click(screen.getByRole('button', { name: 'Save' }))
        await waitFor(() => expect(screen.getByText('Could not save the note')).toBeInTheDocument())
    })
})
