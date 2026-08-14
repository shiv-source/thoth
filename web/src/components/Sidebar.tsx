import { useEffect, useState } from 'react'
import { api, type TreeNode } from '../api/client'

export function Sidebar() {
  const [nodes, setNodes] = useState<TreeNode[]>([])
  const [view, setView] = useState<'search' | 'tree' | 'settings'>('search')

  useEffect(() => {
    api.tree().then((r) => setNodes(r.nodes)).catch(() => setNodes([]))
  }, [])

  return (
    <aside className="flex w-72 flex-col bg-paper-100 dark:bg-night-900">
      <div className="flex items-center gap-2 border-b border-paper-200 px-5 py-4 dark:border-night-800">
        <span className="text-xl">🦉</span>
        <span className="font-display text-xl font-semibold text-ink-900 dark:text-paper-100">Thoth</span>
      </div>
      <nav className="flex gap-1 px-3 py-2 text-sm">
        {(['search', 'tree', 'settings'] as const).map((v) => (
          <button key={v} onClick={() => setView(v)}
            className={`rounded-lg px-3 py-1.5 capitalize transition ${view === v ? 'bg-ink-900 text-paper-100 dark:bg-paper-100 dark:text-ink-900' : 'text-ink-500 hover:text-ink-900 dark:hover:text-paper-100'}`}>
            {v}
          </button>
        ))}
      </nav>
      <div className="flex-1 overflow-y-auto px-3 pb-3">
        {view === 'search' && <SearchPanel />}
        {view === 'tree' && <Tree nodes={nodes} />}
        {view === 'settings' && <SettingsPanel />}
      </div>
    </aside>
  )
}

function Tree({ nodes }: { nodes: TreeNode[] }) {
  return (
    <ul className="space-y-0.5 text-sm">
      {nodes.map((n) => (
        <li key={n.path}>
          <div className="flex items-center gap-1.5 rounded-md px-2 py-1 text-ink-700 hover:bg-paper-200 dark:text-paper-100 dark:hover:bg-night-800">
            <span className="text-xs">{n.is_dir ? '▸' : '·'}</span>
            <span className="truncate">{n.name}</span>
          </div>
          {n.children && n.children.length > 0 && (
            <ul className="ml-4 border-l border-paper-300 pl-2 dark:border-night-700">
              {n.children.map((c) => (
                <li key={c.path} className="truncate rounded-md px-2 py-0.5 text-ink-500 hover:bg-paper-200 dark:hover:bg-night-800">
                  {c.name}
                </li>
              ))}
            </ul>
          )}
        </li>
      ))}
    </ul>
  )
}

import { SearchPanel } from './SearchPanel'
import { SettingsPanel } from './SettingsPanel'
