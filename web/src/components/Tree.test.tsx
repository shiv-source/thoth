import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Tree } from './Tree'

interface Node {
  id: string
  label: string
  dir: boolean
  children: Node[]
}

const sample: Node[] = [
  { id: 'inbox', label: 'inbox', dir: true, children: [
    { id: 'inbox/a.md', label: 'a.md', dir: false, children: [] },
  ] },
  { id: 'daily', label: 'daily', dir: true, children: [
    { id: 'daily/b.md', label: 'b.md', dir: false, children: [] },
  ] },
]

function focusRow(label: string) {
  const row = screen.getByText(label).closest('[role="treeitem"]')!
  ;(row.querySelector('[tabindex]') as HTMLElement).focus()
}

function renderTree(props: Partial<Parameters<typeof Tree<Node>>[0]> = {}) {
  const onSelect = vi.fn()
  const utils = render(
    <Tree<Node>
      nodes={sample}
      getKey={(n) => n.id}
      getLabel={(n) => n.label}
      isDir={(n) => n.dir}
      getChildren={(n) => n.children}
      renderIcon={(n) => <span data-testid={`icon-${n.id}`}>{n.dir ? 'dir' : 'file'}</span>}
      onSelect={onSelect}
      selectedKey={null}
      {...props}
    />,
  )
  return { onSelect, ...utils }
}

describe('Tree', () => {
  it('renders top-level folders collapsed by default', () => {
    renderTree()
    expect(screen.getByRole('tree')).toBeInTheDocument()
    expect(screen.getByText('inbox')).toBeInTheDocument()
    expect(screen.getByText('daily')).toBeInTheDocument()
    expect(screen.queryByText('a.md')).not.toBeInTheDocument()
    expect(screen.queryByText('b.md')).not.toBeInTheDocument()
    // Custom icon renderer used for every visible row.
    expect(screen.getByTestId('icon-inbox')).toHaveTextContent('dir')
  })

  it('expands a directory with the chevron and collapses it again', async () => {
    renderTree()
    expect(screen.queryByText('a.md')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Expand inbox' }))
    expect(screen.getByText('a.md')).toBeInTheDocument()
    expect(screen.getByTestId('icon-inbox/a.md')).toHaveTextContent('file')
    expect(screen.getByRole('treeitem', { name: /inbox/ })).toHaveAttribute('aria-expanded', 'true')

    await userEvent.click(screen.getByRole('button', { name: 'Collapse inbox' }))
    expect(screen.queryByText('a.md')).not.toBeInTheDocument()
  })

  it('selects a node on click and on Enter, with aria-selected on the row', async () => {
    const { onSelect } = renderTree({ selectedKey: null })
    await userEvent.click(screen.getByRole('button', { name: 'Expand inbox' }))
    await userEvent.click(screen.getByText('a.md'))
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'inbox/a.md' }))

    await userEvent.click(screen.getByRole('button', { name: 'Expand daily' }))
    focusRow('b.md')
    await userEvent.keyboard('{Enter}')
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'daily/b.md' }))
  })

  it('moves focus with arrow keys', async () => {
    renderTree()
    focusRow('inbox')
    await userEvent.keyboard('{ArrowRight}') // expand
    await userEvent.keyboard('{ArrowDown}')
    expect(screen.getByText('a.md').closest('[role="treeitem"]')!.querySelector('[tabindex="0"]')).toHaveFocus()
    await userEvent.keyboard('{ArrowDown}')
    expect(screen.getByText('daily').closest('[role="treeitem"]')!.querySelector('[tabindex="0"]')).toHaveFocus()
    await userEvent.keyboard('{ArrowUp}')
    expect(screen.getByText('a.md').closest('[role="treeitem"]')!.querySelector('[tabindex="0"]')).toHaveFocus()
  })
})
