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

})

afterAll(() => { globalThis.WebSocket = original })

describe('ChatPanel thinking status', () => {
  afterEach(() => { FakeWS.instances = [] })

  it('shows what the assistant is thinking and hides it on the first delta', () => {
    renderPanel()
    act(() => FakeWS.instances[0]!.open())
    const ws = FakeWS.instances[0]!
    const emit = (frame: object) =>
      act(() => ws.onmessage?.({ data: JSON.stringify(frame) }))

    emit({ type: 'assistant_start' })
    expect(screen.getByText('Thinking…')).toBeInTheDocument()

    emit({ type: 'assistant_thinking', text: 'checking the inbox folder' })
    expect(screen.getByText(/checking the inbox folder/)).toBeInTheDocument()

    emit({ type: 'assistant_delta', text: 'hi' })
    expect(screen.queryByText(/checking the inbox folder/)).not.toBeInTheDocument()
  })
})
