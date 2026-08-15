import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeAll, describe, expect, it, vi } from 'vitest'
import { Tooltip } from './Tooltip'

// Radix positions tooltips with a ResizeObserver; jsdom does not ship one.
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
beforeAll(() => {
  vi.stubGlobal('ResizeObserver', ResizeObserverMock)
})

describe('Tooltip', () => {
  it('shows the styled bubble on hover with the label', async () => {
    render(
      <Tooltip label="Expand all">
        <button type="button">Toggle</button>
      </Tooltip>,
    )
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    await userEvent.hover(screen.getByRole('button'))
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Expand all')
  })

  it('places the bubble on the requested side and alignment', async () => {
    render(
      <Tooltip label="Delete chat" side="bottom" align="end">
        <button type="button">Delete</button>
      </Tooltip>,
    )
    await userEvent.hover(screen.getByRole('button'))
    const bubble = await screen.findByRole('tooltip')
    expect(bubble).toHaveAttribute('data-side', 'bottom')
    expect(bubble).toHaveAttribute('data-align', 'end')
  })
})
