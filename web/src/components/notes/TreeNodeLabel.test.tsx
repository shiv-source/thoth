import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithStore } from '../../test/renderWithStore'
import { TreeNodeLabel } from './TreeNodeLabel'

describe('TreeNodeLabel', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('truncates a long leaf filename and exposes the full name on hover', async () => {
        const long = 'i-checked-there-s-no-record-of-tuesday-s-standup-in-the-wi.md'
        renderWithStore(<TreeNodeLabel title={long} isLeaf={true} />)

        const text = screen.getByText(long)
        // The text node itself carries the truncate utility (so the wrapper's
        // min-width:0 / overflow:hidden clips it instead of stretching the row).
        expect(text.className).toContain('truncate')

        // Hovering the leaf shows the full filename in the tooltip.
        await userEvent.hover(screen.getByText(long))
        expect(await screen.findByRole('tooltip')).toHaveTextContent(long)
    })

    it('keeps a short leaf filename truncatable but not clipped', () => {
        renderWithStore(<TreeNodeLabel title="short.md" isLeaf={true} />)
        expect(screen.getByText('short.md').className).toContain('truncate')
    })
})
