import { useEffect, useState } from 'react'
import { api, type TreeNode } from '../api/client'
import { SearchPanel } from './SearchPanel'

export function Sidebar({ openPath, onOpenNote }: {
  openPath: string | null
  onOpenNote: (path: string) => void
}) {
  const [nodes, setNodes] = useState<TreeNode[]>([])

  useEffect(() => {
    api.tree().then((r) => setNodes(r.nodes)).catch(() => setNodes([]))
  }, [])

  return (
    <aside className="flex w-72 shrink-0 flex-col border-r border-line bg-app">
      <div className="flex h-14 shrink-0 items-center gap-2.5 px-4">
        <span className="text-xl leading-none">🦉</span>
        <span className="font-display text-lg font-semibold tracking-tight text-heading">Thoth</span>
        <span className="ml-auto h-2 w-2 rounded-full bg-accent" aria-hidden="true" />
      </div>
      <div className="px-3">
        <SearchPanel onOpen={onOpenNote} />
      </div>
      <div className="px-3 pb-1.5 pt-5 text-[11px] font-semibold uppercase tracking-wider text-subtle">
        Wiki
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-3">
        {nodes.length === 0 ? (
          <p className="px-1 text-sm text-subtle">No notes yet — your wiki is empty.</p>
        ) : (
          <Tree nodes={nodes} openPath={openPath} onOpen={onOpenNote} />
        )}
      </div>
    </aside>
  )
}

function Tree({ nodes, openPath, onOpen }: {
  nodes: TreeNode[]
  openPath: string | null
  onOpen: (path: string) => void
}) {
  return (
    <ul className="space-y-px text-sm">
      {nodes.map((n) => (
        <TreeRow key={n.path} node={n} depth={0} openPath={openPath} onOpen={onOpen} />
      ))}
    </ul>
  )
}

function TreeRow({ node, depth, openPath, onOpen }: {
  node: TreeNode
  depth: number
  openPath: string | null
  onOpen: (path: string) => void
}) {
  const [expanded, setExpanded] = useState(node.is_dir && depth === 0)

  if (node.is_dir) {
    return (
      <li>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setExpanded((e) => !e)}
            aria-label={`${expanded ? 'Collapse' : 'Expand'} ${node.name}`}
            className="flex h-6 w-5 shrink-0 items-center justify-center rounded text-[10px] text-subtle transition hover:text-ink"
          >
            {expanded ? '▾' : '▸'}
          </button>
          <span className="truncate py-1 pr-1 font-medium text-ink">{node.name}</span>
        </div>
        {expanded && node.children && node.children.length > 0 && (
          <ul className="ml-2 border-l border-line pl-2">
            {node.children.map((c) => (
              <TreeRow key={c.path} node={c} depth={depth + 1} openPath={openPath} onOpen={onOpen} />
            ))}
          </ul>
        )}
      </li>
    )
  }

  const active = openPath === node.path
  return (
    <li>
      <button
        onClick={() => onOpen(node.path)}
        className={`flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-left transition ${
          active ? 'bg-accent-soft font-medium text-accent' : 'text-ink hover:bg-raised'
        }`}
      >
        <span className="flex w-5 shrink-0 items-center justify-center">
          <DocIcon className={active ? 'text-accent' : 'text-subtle'} />
        </span>
        <span className="truncate">{node.name}</span>
      </button>
    </li>
  )
}

function DocIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.3" aria-hidden="true"
      className={`h-3.5 w-3.5 shrink-0 ${className ?? ''}`}>
      <path d="M4 1.8h5.2L12 4.6v9.6H4z" strokeLinejoin="round" />
      <path d="M9.2 1.8v2.8H12" strokeLinejoin="round" />
    </svg>
  )
}
