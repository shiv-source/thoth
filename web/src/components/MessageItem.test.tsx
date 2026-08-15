import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MessageItem } from './MessageItem'

describe('MessageItem copy', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('copies the assistant message on click and shows the check', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })

    render(<MessageItem message={{ role: 'assistant', content: 'hello **wiki**' }} />)

    await userEvent.click(screen.getByRole('button', { name: 'Copy message' }))
    expect(writeText).toHaveBeenCalledWith('hello **wiki**')
    expect(screen.getByRole('button', { name: 'Copy message' })).toBeInTheDocument()
    // The check icon replaces the copy icon briefly.
    expect(document.querySelector('.lucide-check')).not.toBeNull()

    vi.useFakeTimers()
    try {
      act(() => { vi.advanceTimersByTime(1600) })
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not render a copy button on user messages or while streaming', () => {
    const { rerender } = render(<MessageItem message={{ role: 'user', content: 'hi' }} />)
    expect(screen.queryByRole('button', { name: 'Copy message' })).not.toBeInTheDocument()

    rerender(<MessageItem message={{ role: 'assistant', content: 'hi' }} streaming />)
    expect(screen.queryByRole('button', { name: 'Copy message' })).not.toBeInTheDocument()
  })
})
