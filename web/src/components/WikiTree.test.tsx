import { useState } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { WikiTree } from './WikiTree'

// Harness: WikiTree is controlled from the sidebar now; keep the expanded
// state in the test so interactions behave like production.
function renderWikiTree(openPath: string | null = null) {
  const onOpenNote = vi.fn()
  function Wrapper() {
    const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())
    return (
      <WikiTree
        openPath={openPath}
        onOpenNote={onOpenNote}
        expandedKeys={expandedKeys}
        onExpandedChange={setExpandedKeys}
      />
    )
  }
  return { onOpenNote, ...render(<Wrapper />) }
}

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
    renderWikiTree()

    expect(screen.getByText('Loading…')).toBeInTheDocument()
    // Folders start collapsed: top-level names visible, files hidden.
    expect(await screen.findByText('meetings')).toBeInTheDocument()
    expect(screen.getByText('todos')).toBeInTheDocument()
    expect(screen.queryByText('standup.md')).not.toBeInTheDocument()
  })

  it('opens the clicked note and marks it selected', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    const { onOpenNote } = renderWikiTree()

    await userEvent.click(await screen.findByRole('button', { name: 'Expand meetings' }))
    await userEvent.click(screen.getByText('standup.md'))
    expect(onOpenNote).toHaveBeenCalledWith('meetings/standup.md')
  })

  it('collapsing a folder hides its files', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    renderWikiTree()

    await userEvent.click(await screen.findByRole('button', { name: 'Expand meetings' }))
    expect(screen.getByText('standup.md')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Collapse meetings' }))
    await waitFor(() => expect(screen.queryByText('standup.md')).not.toBeInTheDocument())
  })

  it('shows per-folder note-count badges', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(treeResponse), { status: 200 })))
    renderWikiTree()
    // Badges: one file under each folder (recursive counts).
    expect((await screen.findByText('meetings')).parentElement?.textContent).toContain('1')
    expect(screen.getByText('todos').parentElement?.textContent).toContain('1')
  })

  it('shows an error state when the tree fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('boom', { status: 500 })))
    renderWikiTree()

    expect(await screen.findByText('Could not load the wiki tree')).toBeInTheDocument()
  })
})
