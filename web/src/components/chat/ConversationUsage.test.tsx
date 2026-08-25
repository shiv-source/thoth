import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { TokenUsage } from '../../ws/protocol'
import { ConversationUsage } from './ConversationUsage'

describe('ConversationUsage', () => {
    it('renders the summed input/output counts with a total label', () => {
        const usage: TokenUsage = {
            input_tokens: 12345,
            output_tokens: 678,
            cache_read_tokens: 0,
            cache_write_tokens: 0
        }
        render(<ConversationUsage usage={usage} />)
        expect(screen.getByLabelText('Conversation token usage')).toBeInTheDocument()
        expect(screen.getByLabelText('input tokens')).toBeInTheDocument()
        expect(screen.getByLabelText('output tokens')).toBeInTheDocument()
        expect(screen.getByText('12,345')).toBeInTheDocument()
        expect(screen.getByText('678')).toBeInTheDocument()
    })

    it('appends cache counters only when non-zero', () => {
        const usage: TokenUsage = { input_tokens: 10, output_tokens: 4, cache_read_tokens: 5, cache_write_tokens: 3 }
        render(<ConversationUsage usage={usage} />)
        expect(screen.getByText('5 cache read')).toBeInTheDocument()
        expect(screen.getByText('3 cache write')).toBeInTheDocument()
    })

    it('renders nothing when every counter is zero', () => {
        const usage: TokenUsage = { input_tokens: 0, output_tokens: 0, cache_read_tokens: 0, cache_write_tokens: 0 }
        const { container } = render(<ConversationUsage usage={usage} />)
        expect(container).toBeEmptyDOMElement()
    })
})
