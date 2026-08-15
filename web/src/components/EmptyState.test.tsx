import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithStore } from '../test/renderWithStore'
import { EmptyState } from './EmptyState'

describe('EmptyState', () => {
    it('renders icon, title, and hint', () => {
        renderWithStore(<EmptyState icon="🦉" title="Select a note" hint="Backlinks land later." />)
        expect(screen.getByText('🦉')).toBeInTheDocument()
        expect(screen.getByText('Select a note')).toBeInTheDocument()
        expect(screen.getByText('Backlinks land later.')).toBeInTheDocument()
    })

    it('omits the hint when not provided', () => {
        renderWithStore(<EmptyState icon="🔍" title="Nothing here" />)
        expect(screen.getByText('Nothing here')).toBeInTheDocument()
    })
})
