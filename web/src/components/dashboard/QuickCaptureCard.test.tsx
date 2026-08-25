import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { renderWithStore } from '../../test/renderWithStore'
import { QuickCaptureCard } from './QuickCaptureCard'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

describe('QuickCaptureCard', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it('captures text through the unified endpoint and reports the saved path', async () => {
        const onCaptured = vi.fn()
        stubAPI(mocks, {
            'POST /api/v1/capture': () => ({ path: 'inbox/hello-world.md', title: 'Hello world', type: 'inbox' })
        })
        renderWithStore(<QuickCaptureCard onCaptured={onCaptured} />)

        fireEvent.change(screen.getByRole('textbox', { name: 'Quick capture' }), {
            target: { value: 'Hello world' }
        })
        fireEvent.click(screen.getByRole('button', { name: /Capture/ }))

        await waitFor(() =>
            expect(mocks.post).toHaveBeenCalledWith('/api/v1/capture', {
                kind: 'note',
                text: 'Hello world',
                folder: 'inbox'
            })
        )
        expect(await screen.findByText('Captured to inbox/hello-world.md')).toBeInTheDocument()
        expect(onCaptured).toHaveBeenCalledWith('inbox/hello-world.md')
        // The input clears after a successful capture.
        expect(screen.getByRole('textbox', { name: 'Quick capture' })).toHaveValue('')
    })

    it('does nothing on empty or whitespace-only input', () => {
        renderWithStore(<QuickCaptureCard />)
        fireEvent.change(screen.getByRole('textbox', { name: 'Quick capture' }), { target: { value: '   ' } })
        fireEvent.click(screen.getByRole('button', { name: /Capture/ }))
        expect(mocks.post).not.toHaveBeenCalled()
    })

    it('toasts an error when the server is unreachable', async () => {
        stubAPI(mocks, {}) // every POST rejects
        renderWithStore(<QuickCaptureCard />)
        fireEvent.change(screen.getByRole('textbox', { name: 'Quick capture' }), { target: { value: 'x' } })
        fireEvent.click(screen.getByRole('button', { name: /Capture/ }))
        expect(await screen.findByText(/Could not capture/)).toBeInTheDocument()
    })
})
