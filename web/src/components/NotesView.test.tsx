import { useState } from 'react'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../test/mockAxios'
import { renderWithStore } from '../test/renderWithStore'
import { NotesView } from './NotesView'
import { ToastProvider } from './Toast'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const tree = () => ({
    nodes: [
        {
            name: 'meetings',
            path: 'meetings',
            is_dir: true,
            children: [{ name: 'standup.md', path: 'meetings/standup.md', is_dir: false, children: null }]
        }
    ]
})

function renderNotes(openPath: string | null = null) {
    const onOpenNote = vi.fn()
    const utils = renderWithStore(
        <ToastProvider>
            <NotesView openPath={openPath} onOpenNote={onOpenNote} onOpenSettings={vi.fn()} />
        </ToastProvider>
    )
    return { onOpenNote, ...utils }
}

describe('NotesView', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        mocks.get.mockImplementation((url: string) => {
            if (url === '/api/wiki/tree') return Promise.resolve({ data: tree() })
            return Promise.reject(new Error(`unhandled ${url}`))
        })
    })

    it('shows the empty state until a note is selected', async () => {
        renderNotes()
        expect(await screen.findByText('Select a note to read it here')).toBeInTheDocument()
    })

    it('expand-all reveals every folder; the same toggle collapses them again', async () => {
        renderNotes()
        await userEvent.click(await screen.findByRole('button', { name: 'Expand all folders' }))
        expect(screen.getByText('standup.md')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Collapse all folders' })).toBeInTheDocument()

        await userEvent.click(screen.getByRole('button', { name: 'Collapse all folders' }))
        expect(screen.queryByText('standup.md')).not.toBeInTheDocument()
    })

    it('expands the open note’s ancestor folders automatically', async () => {
        mocks.get.mockImplementation((url: string) => {
            if (url === '/api/wiki/tree') {
                return Promise.resolve({
                    data: {
                        nodes: [
                            {
                                name: 'meetings',
                                path: 'meetings',
                                is_dir: true,
                                children: [
                                    {
                                        name: 'standup.md',
                                        path: 'meetings/standup.md',
                                        is_dir: false,
                                        children: null
                                    }
                                ]
                            }
                        ]
                    }
                })
            }
            if (url === '/api/notes?path=meetings%2Fstandup.md') {
                return Promise.resolve({ data: { path: 'meetings/standup.md', content: '# Standup' } })
            }
            return Promise.reject(new Error(`unhandled ${url}`))
        })
        const onOpenNote = vi.fn()
        renderWithStore(
            <ToastProvider>
                <NotesView openPath="meetings/standup.md" onOpenNote={onOpenNote} onOpenSettings={vi.fn()} />
            </ToastProvider>
        )
        // No expand-all click needed: the ancestor folder is already open.
        expect(await screen.findByText('standup.md')).toBeInTheDocument()
    })

    it('opens the selected note inline and closes back to the empty state', async () => {
        mocks.get.mockImplementation((url: string) => {
            if (url === '/api/wiki/tree') return Promise.resolve({ data: tree() })
            if (url === '/api/notes?path=meetings%2Fstandup.md') {
                return Promise.resolve({ data: { path: 'meetings/standup.md', content: '# Standup\n\nnotes here' } })
            }
            return Promise.reject(new Error(`unhandled ${url}`))
        })
        // A stateful wrapper mirrors App: onOpenNote(null) clears openPath.
        function Wrapper() {
            const [open, setOpen] = useState<string | null>('meetings/standup.md')
            return (
                <ToastProvider>
                    <NotesView openPath={open} onOpenNote={setOpen} onOpenSettings={vi.fn()} />
                </ToastProvider>
            )
        }
        renderWithStore(<Wrapper />)

        expect(await screen.findByText('Standup')).toBeInTheDocument()
        expect(screen.queryByText('Select a note to read it here')).not.toBeInTheDocument()

        await userEvent.click(screen.getByRole('button', { name: 'Close note' }))
        await waitFor(() => expect(screen.queryByText('Standup')).not.toBeInTheDocument())
        expect(screen.getByText('Select a note to read it here')).toBeInTheDocument()
    })
})
