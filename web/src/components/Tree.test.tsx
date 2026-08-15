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
    {
        id: 'inbox',
        label: 'inbox',
        dir: true,
        children: [{ id: 'inbox/a.md', label: 'a.md', dir: false, children: [] }]
    },
    {
        id: 'daily',
        label: 'daily',
        dir: true,
        children: [{ id: 'daily/b.md', label: 'b.md', dir: false, children: [] }]
    }
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
        />
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

    it('toggles a folder by clicking its row', async () => {
        renderTree()
        await userEvent.click(screen.getByText('inbox'))
        expect(screen.getByText('a.md')).toBeInTheDocument()
        await userEvent.click(screen.getByText('inbox'))
        expect(screen.queryByText('a.md')).not.toBeInTheDocument()
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

    it('re-renders only the affected rows when the selection moves', () => {
        // Stable accessors and callbacks are the contract the memoized rows rely
        // on — inline closures would re-render every row regardless.
        const renderIcon = vi.fn((n: Node) => <span data-testid={`icon-${n.id}`}>{n.dir ? 'dir' : 'file'}</span>)
        const onSelect = vi.fn()
        const onExpandedChange = vi.fn()
        const getKey = (n: Node) => n.id
        const getLabel = (n: Node) => n.label
        const isDir = (n: Node) => n.dir
        const getChildren = (n: Node) => n.children
        const expandedKeys = new Set(['inbox', 'daily'])
        const props = {
            nodes: sample,
            getKey,
            getLabel,
            isDir,
            getChildren,
            renderIcon,
            onSelect,
            expandedKeys,
            onExpandedChange,
            selectedKey: null as string | null
        }

        const { rerender } = render(<Tree<Node> {...props} />)
        // All four rows visible: inbox, a.md, daily, b.md.
        expect(screen.getByText('a.md')).toBeInTheDocument()
        expect(screen.getByText('b.md')).toBeInTheDocument()
        const calls = renderIcon.mock.calls.length

        rerender(<Tree<Node> {...props} selectedKey="inbox/a.md" />)
        expect(screen.getByText('a.md').closest('[role="treeitem"]')).toHaveAttribute('aria-selected', 'true')
        // Only the newly selected row re-rendered — the other three skipped.
        expect(renderIcon.mock.calls.length).toBe(calls + 1)
    })
})

describe('Tree controlled mode', () => {
    it('defers expansion to the caller via expandedKeys/onExpandedChange', async () => {
        let keys = new Set<string>(['inbox'])
        const onExpandedChange = vi.fn((next: Set<string>) => {
            keys = next
        })
        const { rerender } = render(
            <Tree<Node>
                nodes={sample}
                getKey={(n) => n.id}
                getLabel={(n) => n.label}
                isDir={(n) => n.dir}
                getChildren={(n) => n.children}
                renderIcon={() => null}
                onSelect={() => {}}
                selectedKey={null}
                expandedKeys={keys}
                onExpandedChange={onExpandedChange}
            />
        )
        expect(screen.getByText('a.md')).toBeInTheDocument()
        expect(screen.queryByText('b.md')).not.toBeInTheDocument()

        // Collapsing goes through the callback.
        await userEvent.click(screen.getByRole('button', { name: 'Collapse inbox' }))
        expect(onExpandedChange).toHaveBeenCalled()
        const collapsed = onExpandedChange.mock.calls.at(-1)![0]
        expect(collapsed.has('inbox')).toBe(false)

        // Rerendering with the collapsed set hides the children.
        rerender(
            <Tree<Node>
                nodes={sample}
                getKey={(n) => n.id}
                getLabel={(n) => n.label}
                isDir={(n) => n.dir}
                getChildren={(n) => n.children}
                renderIcon={() => null}
                onSelect={() => {}}
                selectedKey={null}
                expandedKeys={collapsed}
                onExpandedChange={onExpandedChange}
            />
        )
        expect(screen.queryByText('a.md')).not.toBeInTheDocument()
    })
})
