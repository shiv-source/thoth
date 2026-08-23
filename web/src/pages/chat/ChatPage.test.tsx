// mockAxios must initialize before any module that transitively imports
// axios — the hoisted vi.mock factory closes over axiosModuleMock, so a
// later-imported helper would be in the TDZ when the factory runs.
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ChatPage } from './ChatPage'
import { FakeWS } from '../../test/fakeWS'
import { renderWithStore } from '../../test/renderWithStore'

const original = globalThis.WebSocket
globalThis.WebSocket = FakeWS as unknown as typeof WebSocket

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

function renderPanel() {
    return renderWithStore(<ChatPage onOpenSettings={() => {}} onOpenNote={() => {}} />)
}

describe('ChatPage', () => {
    beforeEach(() => {
        stubAPI(mocks, { 'GET /api/v1/conversations': () => ({ conversations: [] }) })
    })
    afterEach(() => {
        FakeWS.instances = []
        // turn_done pushes /chat/<id> onto the URL without firing popstate;
        // reset it so the next test does not mount onto a stale route.
        window.history.pushState(null, '', '/')
    })

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

    it('lays messages out with a 16px gap on the antd Flex container', async () => {
        const { container } = renderPanel()
        act(() => FakeWS.instances[0]!.open())
        await userEvent.type(screen.getByPlaceholderText(/Ask your wiki/), 'hello')
        await userEvent.click(screen.getByRole('button', { name: /Send/ }))
        // antd Flex resets its children's margins (and its own padding), so
        // spacing must live on the inner Flex's own gap — regression guard
        // for the space-y-* bug.
        const scroller = container.querySelector('.overflow-y-auto')
        expect(scroller).not.toBeNull()
        expect(scroller!.querySelector('.ant-flex')?.getAttribute('style')).toContain('gap: 16px')
    })

    it('shows the tool status line while a tool runs and hides it on turn_done', () => {
        renderPanel()
        act(() => FakeWS.instances[0]!.open())
        const emit = (frame: string) => act(() => FakeWS.instances[0]!.onmessage?.({ data: frame }))

        emit(
            JSON.stringify({
                type: 'tool_activity',
                tool: 'Read',
                detail: JSON.stringify({ path: 'meetings/standup.md' })
            })
        )
        expect(screen.getByText(/meetings\/standup\.md/)).toBeInTheDocument()

        emit(JSON.stringify({ type: 'turn_done' }))
        expect(screen.queryByText(/meetings\/standup\.md/)).not.toBeInTheDocument()
    })

    it('refreshes the conversation list when a chat is created', async () => {
        renderPanel()
        act(() => FakeWS.instances[0]!.open())
        const emit = (frame: object) => act(() => FakeWS.instances[0]!.onmessage?.({ data: JSON.stringify(frame) }))

        emit({ type: 'turn_done', conversation_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1' })
        await waitFor(() => expect(mocks.get).toHaveBeenCalledWith('/api/v1/conversations'))
    })

    it('shows token usage under the last message when turn_done carries it', () => {
        renderPanel()
        act(() => FakeWS.instances[0]!.open())
        const emit = (frame: object) => act(() => FakeWS.instances[0]!.onmessage?.({ data: JSON.stringify(frame) }))

        emit({ type: 'assistant_delta', text: 'the answer' })
        emit({
            type: 'turn_done',
            conversation_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1',
            usage: { input_tokens: 10, output_tokens: 4, cache_read_tokens: 5, cache_write_tokens: 0 }
        })
        expect(screen.getByText('10 in · 4 out · 5 cache read')).toBeInTheDocument()
    })

    it('renders no usage line when turn_done carries none', () => {
        renderPanel()
        act(() => FakeWS.instances[0]!.open())
        const emit = (frame: object) => act(() => FakeWS.instances[0]!.onmessage?.({ data: JSON.stringify(frame) }))

        emit({ type: 'assistant_delta', text: 'the answer' })
        emit({ type: 'turn_done', conversation_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1' })
        expect(screen.queryByLabelText('Token usage')).not.toBeInTheDocument()
    })
})

afterAll(() => {
    globalThis.WebSocket = original
})

describe('ChatPage thinking status', () => {
    afterEach(() => {
        FakeWS.instances = []
    })

    it('shows what the assistant is thinking and hides it on the first delta', () => {
        renderPanel()
        act(() => FakeWS.instances[0]!.open())
        const ws = FakeWS.instances[0]!
        const emit = (frame: object) => act(() => ws.onmessage?.({ data: JSON.stringify(frame) }))

        emit({ type: 'assistant_start' })
        expect(screen.getByText('Thinking…')).toBeInTheDocument()

        emit({ type: 'assistant_thinking', text: 'checking the inbox folder' })
        expect(screen.getByText(/checking the inbox folder/)).toBeInTheDocument()

        emit({ type: 'assistant_delta', text: 'hi' })
        expect(screen.queryByText(/checking the inbox folder/)).not.toBeInTheDocument()
    })
})
