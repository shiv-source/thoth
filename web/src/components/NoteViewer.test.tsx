import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../test/mockAxios'
import { renderWithStore } from '../test/renderWithStore'
import { ToastProvider } from './Toast'
import { NoteViewer } from './NoteViewer'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

function renderViewer(inline = false) {
    const onClose = vi.fn()
    const utils = renderWithStore(
        <ToastProvider>
            <NoteViewer path="knowledge/note.md" onClose={onClose} inline={inline} />
        </ToastProvider>
    )
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

    it('renders inline (fills its container, no fixed drawer) when inline is set', async () => {
        const { container } = renderViewer(true)
        await screen.findByText('Hello')
        const aside = container.querySelector('aside')
        expect(aside?.className).toContain('flex-1')
        expect(aside?.className).not.toContain('fixed')
    })

    it('renders as the fixed right drawer by default', async () => {
        const { container } = renderViewer()
        await screen.findByText('Hello')
        expect(container.querySelector('aside')?.className).toContain('fixed')
    })

    it('closes via the close button', async () => {
        const { onClose } = renderViewer(true)
        await userEvent.click(screen.getByRole('button', { name: 'Close note' }))
        expect(onClose).toHaveBeenCalled()
    })

    it('copies the raw note and shows the toast', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        Object.assign(navigator, { clipboard: { writeText } })
        renderViewer(true)
        await screen.findByText('Hello')
        await userEvent.click(screen.getByRole('button', { name: 'Copy raw' }))
        expect(writeText).toHaveBeenCalledWith('# Hello\n\nbody')
        expect(await screen.findByText('Note copied to clipboard')).toBeInTheDocument()
    })
})
