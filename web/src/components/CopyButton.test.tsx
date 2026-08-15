import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderWithStore } from '../test/renderWithStore'
import { CopyButton } from './CopyButton'
import { ToastProvider } from './Toast'

function renderCopy(text = 'hello', toast?: string) {
    return renderWithStore(
        <ToastProvider>
            <CopyButton text={text} label="Copy code" toast={toast} />
        </ToastProvider>
    )
}

describe('CopyButton', () => {
    it('copies the text and flips to the check icon', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        Object.assign(navigator, { clipboard: { writeText } })
        renderCopy('const x = 1')
        await userEvent.click(screen.getByRole('button', { name: 'Copy code' }))
        expect(writeText).toHaveBeenCalledWith('const x = 1')
        expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
    })

    it('surfaces the toast when a message is given', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        Object.assign(navigator, { clipboard: { writeText } })
        renderCopy('x', 'Code copied to clipboard')
        await userEvent.click(screen.getByRole('button', { name: 'Copy code' }))
        expect(await screen.findByText('Code copied to clipboard')).toBeInTheDocument()
    })

    it('does nothing when the clipboard is unavailable', async () => {
        Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } })
        renderCopy('x')
        await userEvent.click(screen.getByRole('button', { name: 'Copy code' }))
        expect(screen.getByRole('button', { name: 'Copy code' })).toBeInTheDocument()
    })
})
