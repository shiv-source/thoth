import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { WikiTree } from './WikiTree'

const treeResponse = {
  nodes: [
    { name: 'meetings', path: 'meetings', is_dir: true, children: [
      { name: 'standup.md', path: 'meetings/standup.md', is_dir: false, children: null },
    ] },
    { name: 'todos', path: 'todos', is_dir: true, children: [
      { name: 'TODO.md', path: 'todos/TODO.md', is_dir: false, children: null },
    ] },
  ],
}

describe('WikiTree', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('renders the nested wiki structure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    render(<WikiTree openPath={null} onOpenNote={() => {}} />)

    expect(screen.getByText('Loading…')).toBeInTheDocument()
    // Folders start collapsed: top-level names visible, files hidden.
    expect(await screen.findByText('meetings')).toBeInTheDocument()
    expect(screen.getByText('todos')).toBeInTheDocument()
    expect(screen.queryByText('standup.md')).not.toBeInTheDocument()
  })

  it('opens the clicked note and shows it as selected', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    const onOpenNote = vi.fn()
    const { rerender } = render(<WikiTree openPath={null} onOpenNote={onOpenNote} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Expand meetings' }))
    await userEvent.click(screen.getByText('standup.md'))
    expect(onOpenNote).toHaveBeenCalledWith('meetings/standup.md')

    rerender(<WikiTree openPath="meetings/standup.md" onOpenNote={onOpenNote} />)
    expect(screen.getByText('standup.md').closest('[role="treeitem"]')).toHaveAttribute('aria-selected', 'true')
  })

  it('collapsing a folder hides its files', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    render(<WikiTree openPath={null} onOpenNote={() => {}} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Expand meetings' }))
    expect(screen.getByText('standup.md')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Collapse meetings' }))
    await waitFor(() => expect(screen.queryByText('standup.md')).not.toBeInTheDocument())
  })

  it('shows an error state when the tree fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('boom', { status: 500 })))
    render(<WikiTree openPath={null} onOpenNote={() => {}} />)

    expect(await screen.findByText('Could not load the wiki tree')).toBeInTheDocument()
  })
})

describe('WikiTree controls', () => {
  it('expand-all reveals every folder, collapse-all hides them, badges count files', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    render(<WikiTree openPath={null} onOpenNote={() => {}} />)

    // Badges: one file under each folder (recursive counts).
    expect((await screen.findByText('meetings')).parentElement?.textContent).toContain('1')

    const toggle = screen.getByRole('button', { name: 'Expand all folders' })
    await userEvent.click(toggle)
    expect(screen.getByText('standup.md')).toBeInTheDocument()
    expect(screen.getByText('TODO.md')).toBeInTheDocument()
    // The same button flips to collapse-all.
    expect(screen.getByRole('button', { name: 'Collapse all folders' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Collapse all folders' }))
    expect(screen.queryByText('standup.md')).not.toBeInTheDocument()
    expect(screen.queryByText('TODO.md')).not.toBeInTheDocument()
  })
})
