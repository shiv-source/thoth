import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, afterEach, describe, expect, it } from 'vitest'
import { ChatPanel } from './ChatPanel'
import { FakeWS } from '../test/fakeWS'

const original = globalThis.WebSocket
globalThis.WebSocket = FakeWS as unknown as typeof WebSocket

describe('ChatPanel', () => {
  afterEach(() => { FakeWS.instances = [] })

  it('renders the empty state and sends a message', async () => {
    render(<ChatPanel />)
    // The panel creates its socket in an effect; complete the handshake so
    // the send does not throw (FakeWS models the CONNECTING state).
    act(() => FakeWS.instances[0]!.open())
    expect(screen.getByText(/Ask anything/)).toBeInTheDocument()
    await userEvent.type(screen.getByPlaceholderText(/Ask your wiki/), 'hello')
    await userEvent.click(screen.getByRole('button', { name: /Send/ }))
    expect(screen.getByText('hello')).toBeInTheDocument()
  })
})

afterAll(() => { globalThis.WebSocket = original })
