import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { renderWithStore } from '../../test/renderWithStore'
import { NoteViewer } from './NoteViewer'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

function renderViewer() {
    const onClose = vi.fn()
    const utils = renderWithStore(<NoteViewer path="knowledge/note.md" onClose={onClose} />)
    return { onClose, ...utils }
}

describe('NoteViewer', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        stubAPI(mocks, {
            'GET /api/notes?path=knowledge%2Fnote.md': () => ({ path: 'knowledge/note.md', content: '# Hello\n\nbody' })
        })
    })

    it('renders the note content', async () => {
        renderViewer()
        expect(await screen.findByText('Hello')).toBeInTheDocument()
    })

    it('fills its container (no fixed drawer — the inline reader is the only mode)', async () => {
        const { container } = renderViewer()
        await screen.findByText('Hello')
        const aside = container.querySelector('aside')
        expect(aside?.className).toContain('flex-1')
        expect(aside?.className).not.toContain('fixed')
    })

    it('closes via the close button', async () => {
        const { onClose } = renderViewer()
        await userEvent.click(screen.getByRole('button', { name: 'Close note' }))
        expect(onClose).toHaveBeenCalled()
    })

    it('closes on Escape', async () => {
        const { onClose } = renderViewer()
        await screen.findByText('Hello')
        await userEvent.keyboard('{Escape}')
        expect(onClose).toHaveBeenCalled()
    })

    it('copies the raw note and shows the toast', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        Object.assign(navigator, { clipboard: { writeText } })
        renderViewer()
        await screen.findByText('Hello')
        await userEvent.click(screen.getByRole('button', { name: 'Copy raw' }))
        expect(writeText).toHaveBeenCalledWith('# Hello\n\nbody')
        expect(await screen.findByText('Note copied to clipboard')).toBeInTheDocument()
    })

    it('shows a cannot-preview state for non-previewable attachments without fetching', async () => {
        renderWithStore(<NoteViewer path="attachments/install.sh" onClose={vi.fn()} />)
        expect(await screen.findByText("This file type can't be previewed.")).toBeInTheDocument()
        expect(mocks.get).not.toHaveBeenCalled()
    })

    it('renders an image attachment inline without fetching JSON', async () => {
        renderWithStore(<NoteViewer path="attachments/logo.png" onClose={vi.fn()} />)
        const img = await screen.findByRole('img', { name: 'attachments/logo.png' })
        expect(img).toHaveAttribute('src', '/api/notes?path=attachments%2Flogo.png')
        expect(mocks.get).not.toHaveBeenCalled()
    })

    it('offers a download for non-image attachments', () => {
        renderWithStore(<NoteViewer path="attachments/install.sh" onClose={vi.fn()} />)
        const download = screen.getByRole('link', { name: 'Download' })
        expect(download).toHaveAttribute('href', '/api/notes?path=attachments%2Finstall.sh')
        expect(download).toHaveAttribute('download', 'install.sh')
        expect(mocks.get).not.toHaveBeenCalled()
    })

    it('previews uppercase and long markdown extensions as notes', async () => {
        stubAPI(mocks, {
            'GET /api/notes?path=knowledge%2Fnote.MD': () => ({ path: 'knowledge/note.MD', content: '# Upper' })
        })
        renderWithStore(<NoteViewer path="knowledge/note.MD" onClose={vi.fn()} />)
        expect(await screen.findByText('Upper')).toBeInTheDocument()
    })
})
