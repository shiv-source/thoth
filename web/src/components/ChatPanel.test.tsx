import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, afterEach, describe, expect, it } from 'vitest'
import { ChatPanel } from './ChatPanel'
import { ToastProvider } from './Toast'
import { FakeWS } from '../test/fakeWS'

const original = globalThis.WebSocket
globalThis.WebSocket = FakeWS as unknown as typeof WebSocket

function renderPanel() {
  return render(<ToastProvider><ChatPanel /></ToastProvider>)
}

describe('ChatPanel', () => {
  afterEach(() => { FakeWS.instances = [] })

  it('renders the empty state and sends a message', async () => {
    renderPanel()
    // The panel creates its socket in an effect; complete the handshake so
    // the send does not throw (FakeWS models the CONNECTING state).
    act(() => FakeWS.instances[0]!.open())
    expect(screen.getByText(/Ask anything/)).toBeInTheDocument()
    await userEvent.type(screen.getByPlaceholderText(/Ask your wiki/), 'hello')
    await userEvent.click(screen.getByRole('button', { name: /Send/ }))
    // The bubble carries the text (the top-bar title echoes the first message too).
    expect(screen.getByText('hello', { selector: 'p' })).toBeInTheDocument()
  })

  it('shows the tool status line while a tool runs and hides it on turn_done', () => {
    renderPanel()
    act(() => FakeWS.instances[0]!.open())
    const emit = (frame: string) =>
      act(() => FakeWS.instances[0]!.onmessage?.({ data: frame }))

    emit(JSON.stringify({ type: 'tool_activity', tool: 'Read', detail: JSON.stringify({ path: 'meetings/standup.md' }) }))
    expect(screen.getByText(/meetings\/standup\.md/)).toBeInTheDocument()

    emit(JSON.stringify({ type: 'turn_done' }))
    expect(screen.queryByText(/meetings\/standup\.md/)).not.toBeInTheDocument()
  })

  it('New chat clears the conversation locally', async () => {
    renderPanel()
    act(() => FakeWS.instances[0]!.open())
    await userEvent.type(screen.getByPlaceholderText(/Ask your wiki/), 'hello')
    await userEvent.click(screen.getByRole('button', { name: /Send/ }))
    expect(screen.getByText('hello', { selector: 'p' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /New chat/ }))
    expect(screen.queryByText('hello', { selector: 'p' })).not.toBeInTheDocument()
    expect(screen.getByText('New conversation')).toBeInTheDocument()
  })
})

afterAll(() => { globalThis.WebSocket = original })
