import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MessageItem } from './MessageItem'
import { renderWithStore } from '../../test/renderWithStore'

describe('MessageItem copy', () => {
    afterEach(() => vi.unstubAllGlobals())

    it('copies the assistant message on click and shows the check', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })

        renderWithStore(<MessageItem message={{ role: 'assistant', content: 'hello **wiki**' }} />)

        // Fake timers from the start so the 2s feedback timer is controllable;
        // fireEvent (not userEvent) keeps the click free of internal timer waits.
        vi.useFakeTimers()
        try {
            fireEvent.click(screen.getByRole('button', { name: 'Copy message' }))
            await act(async () => {
                await Promise.resolve()
            })
            expect(writeText).toHaveBeenCalledWith('hello **wiki**')
            // The check replaces the copy icon; the aria-label flips to Copied.
            expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
            expect(document.querySelector('.anticon-check')).not.toBeNull()

            act(() => {
                vi.advanceTimersByTime(2100)
            })
        } finally {
            vi.useRealTimers()
        }
        // The check flips back once the feedback timer expires.
        expect(screen.getByRole('button', { name: 'Copy message' })).toBeInTheDocument()
    })

    it('does not render a copy button on user messages or while streaming', () => {
        const { rerender } = render(<MessageItem message={{ role: 'user', content: 'hi' }} />)
        expect(screen.queryByRole('button', { name: 'Copy message' })).not.toBeInTheDocument()

        rerender(<MessageItem message={{ role: 'assistant', content: 'hi' }} streaming />)
        expect(screen.queryByRole('button', { name: 'Copy message' })).not.toBeInTheDocument()
    })
})
