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

describe('MessageItem header', () => {
    it('renders duration, token usage and actions in the assistant header', () => {
        renderWithStore(
            <MessageItem
                message={{
                    role: 'assistant',
                    content: 'answer',
                    usage: { input_tokens: 120, output_tokens: 45, cache_read_tokens: 0, cache_write_tokens: 0 },
                    durationSecs: 14.256
                }}
            />
        )

        expect(screen.getByText('14.26s')).toBeInTheDocument()
        expect(screen.getByLabelText('Token usage')).toBeInTheDocument()
        expect(screen.getByLabelText('input tokens')).toBeInTheDocument()
        expect(screen.getByLabelText('output tokens')).toBeInTheDocument()
        expect(screen.getByText('120')).toBeInTheDocument()
        expect(screen.getByText('45')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Copy message' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Save as note' })).toBeInTheDocument()
    })

    it('trims trailing zeros from the duration', () => {
        renderWithStore(
            <MessageItem
                message={{
                    role: 'assistant',
                    content: 'answer',
                    durationSecs: 12.5
                }}
            />
        )

        expect(screen.getByText('12.5s')).toBeInTheDocument()
        expect(screen.queryByText('12.50s')).not.toBeInTheDocument()
    })

    it('shows the actions header without duration or usage when the message has none', () => {
        renderWithStore(<MessageItem message={{ role: 'assistant', content: 'no counters' }} />)

        expect(screen.queryByLabelText('Turn duration')).not.toBeInTheDocument()
        expect(screen.queryByLabelText('Token usage')).not.toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Copy message' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Save as note' })).toBeInTheDocument()
    })
})

describe('MessageItem note chips', () => {
    it('renders cited note paths as clickable chips and plain code unchanged', () => {
        const onOpenNote = vi.fn()
        renderWithStore(
            <MessageItem
                message={{ role: 'assistant', content: 'See `links/bookmarks.md` or run `npm install`' }}
                onOpenNote={onOpenNote}
            />
        )

        expect(screen.getByRole('button', { name: 'links/bookmarks.md' })).toBeInTheDocument()
        const plain = screen.getByText('npm install')
        expect(plain.tagName).toBe('CODE')
        expect(screen.queryByRole('button', { name: 'npm install' })).not.toBeInTheDocument()
    })

    it('opens the note when a chip is clicked', () => {
        const onOpenNote = vi.fn()
        renderWithStore(
            <MessageItem
                message={{ role: 'assistant', content: 'Ref: `projects/foo/project.md`' }}
                onOpenNote={onOpenNote}
            />
        )

        fireEvent.click(screen.getByRole('button', { name: 'projects/foo/project.md' }))
        expect(onOpenNote).toHaveBeenCalledWith('projects/foo/project.md')
    })

    it('keeps backticks and HTML inside the path text inert', () => {
        renderWithStore(
            <MessageItem
                message={{ role: 'assistant', content: '``<img src=x onerror=alert(1)>`` and ``a`b.md``' }}
                onOpenNote={vi.fn()}
            />
        )

        // HTML inside the code span stays escaped text, never a live element.
        expect(document.querySelector('img[src]')).toBeNull()
        expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeInTheDocument()
        // A path carrying a backtick is not a note path: plain code, no chip.
        expect(screen.queryByRole('button', { name: 'a`b.md' })).not.toBeInTheDocument()
    })
})
