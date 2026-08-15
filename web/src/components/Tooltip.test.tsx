import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { Tooltip } from './Tooltip'

describe('Tooltip', () => {
  it('renders the label as a tooltip bubble', () => {
    render(
      <Tooltip label="Expand all">
        <button type="button">Toggle</button>
      </Tooltip>,
    )
    expect(screen.getByRole('tooltip')).toHaveTextContent('Expand all')
  })

  it('becomes visible on hover', async () => {
    render(
      <Tooltip label="Expand all">
        <button type="button">Toggle</button>
      </Tooltip>,
    )
    const tooltip = screen.getByRole('tooltip')
    expect(tooltip).toHaveClass('opacity-0')
    await userEvent.hover(screen.getByRole('button'))
    expect(tooltip).toHaveClass('group-hover:opacity-100')
  })
})
