import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DevBanner } from './DevBanner'
import { renderWithStore } from '../test/renderWithStore'

describe('DevBanner', () => {
    it('shows the dev-mode strip when dev is true', () => {
        renderWithStore(<DevBanner dev commit="abc1234" />)
        expect(screen.getByText(/dev mode/i)).toBeInTheDocument()
        expect(screen.getByText(/~\/\.thoth\/dev/)).toBeInTheDocument()
    })

    it('shows the commit id in the strip', () => {
        renderWithStore(<DevBanner dev commit="abc1234" />)
        expect(screen.getByText(/abc1234/)).toBeInTheDocument()
    })

    it('omits the commit when none is known', () => {
        renderWithStore(<DevBanner dev commit="" />)
        expect(screen.queryByText(/·/)).not.toBeInTheDocument()
    })

    it('renders nothing when dev is false', () => {
        renderWithStore(<DevBanner dev={false} commit="" />)
        expect(screen.queryByText(/dev mode/i)).not.toBeInTheDocument()
    })
})
