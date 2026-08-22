import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { TokenUsage } from '../../ws/chat'
import { UsageLine } from './UsageLine'

describe('UsageLine', () => {
    it('renders the input/output counts when usage is present', () => {
        const usage: TokenUsage = { input_tokens: 120, output_tokens: 45, cache_read_tokens: 0, cache_write_tokens: 0 }
        render(<UsageLine usage={usage} />)
        expect(screen.getByText('120 in · 45 out')).toBeInTheDocument()
        expect(screen.getByLabelText('Token usage')).toBeInTheDocument()
    })

    it('appends cache counters only when non-zero', () => {
        const usage: TokenUsage = { input_tokens: 10, output_tokens: 4, cache_read_tokens: 5, cache_write_tokens: 3 }
        render(<UsageLine usage={usage} />)
        expect(screen.getByText('10 in · 4 out · 5 cache read · 3 cache write')).toBeInTheDocument()
    })

    it('renders nothing when usage is absent', () => {
        const { container } = render(<UsageLine usage={null} />)
        expect(container).toBeEmptyDOMElement()
    })

    it('renders nothing when every counter is zero', () => {
        const usage: TokenUsage = { input_tokens: 0, output_tokens: 0, cache_read_tokens: 0, cache_write_tokens: 0 }
        const { container } = render(<UsageLine usage={usage} />)
        expect(container).toBeEmptyDOMElement()
    })
})
