import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ToastProvider, useToast } from './Toast'

function Harness() {
  const { toast } = useToast()
  return <button onClick={() => toast('Saved settings', 'success')}>fire</button>
}

function ErrorHarness() {
  const { toast } = useToast()
  return <button onClick={() => toast('Connection lost', 'error')}>fire error</button>
}

describe('ToastProvider', () => {
  afterEach(() => { vi.useRealTimers() })

  it('renders a toast on call and auto-dismisses after 3 seconds', () => {
    vi.useFakeTimers()
    render(<ToastProvider><Harness /></ToastProvider>)

    fireEvent.click(screen.getByRole('button', { name: 'fire' }))
    const toast = screen.getByText('Saved settings')
    expect(toast).toBeInTheDocument()
    // success toasts carry the emerald dot
    expect(toast.querySelector('span')).toHaveClass('bg-emerald-500')

    act(() => { vi.advanceTimersByTime(3000) })
    expect(screen.queryByText('Saved settings')).not.toBeInTheDocument()
  })

  it('renders error toasts with the red dot', () => {
    vi.useFakeTimers()
    render(<ToastProvider><ErrorHarness /></ToastProvider>)
    fireEvent.click(screen.getByRole('button', { name: 'fire error' }))
    expect(screen.getByText('Connection lost').querySelector('span')).toHaveClass('bg-red-500')
  })

  it('closes on click', () => {
    vi.useFakeTimers()
    render(<ToastProvider><Harness /></ToastProvider>)
    fireEvent.click(screen.getByRole('button', { name: 'fire' }))
    fireEvent.click(screen.getByText('Saved settings'))
    expect(screen.queryByText('Saved settings')).not.toBeInTheDocument()
  })
})
