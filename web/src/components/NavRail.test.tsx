import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { renderWithStore } from '../test/renderWithStore'
import { NavRail } from './NavRail'

describe('NavRail', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('renders every view button and highlights the active one', () => {
        window.history.pushState(null, '', '/notes')
        renderWithStore(<NavRail />)
        expect(screen.getByRole('button', { name: 'Chat' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notes' })).toHaveAttribute('aria-current', 'page')
        expect(screen.getByRole('button', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
        expect(screen.getByRole('button', { name: 'Search' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Settings' })).toBeInTheDocument()
    })

    it('navigates views through the path on click', async () => {
        renderWithStore(<NavRail />)
        await userEvent.click(screen.getByRole('button', { name: 'Dashboard' }))
        expect(window.location.pathname).toBe('/')
        expect(screen.getByRole('button', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page')
    })
})
