import { useEffect, useMemo, useState } from 'react'
import { ChevronsDownUp, ChevronsUpDown, FileText, Folder, FolderOpen } from 'lucide-react'
import { api, type TreeNode } from '../api/client'
import { Tree } from './Tree'

// WikiTree renders the wiki directory as a folder tree via the reusable
// Tree component: folders collapse with chevrons, files open the note.
// Expand/collapse-all buttons and per-folder note counts live in its header.
export function WikiTree({ openPath, onOpenNote }: { openPath: string | null; onOpenNote: (path: string) => void }) {
  const [nodes, setNodes] = useState<TreeNode[] | null>(null)
  const [error, setError] = useState(false)
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())

  useEffect(() => {
    api.tree()
      .then((r) => setNodes(r.nodes))
      .catch(() => setError(true))
  }, [])

  // One pass: every dir key plus the recursive file count per dir.
  const { allDirs, fileCounts } = useMemo(() => {
    const dirs = new Set<string>()
    const counts = new Map<string, number>()
    const walk = (list: TreeNode[]): number => {
      let files = 0
      for (const n of list) {
        if (n.is_dir) {
          dirs.add(n.path)
          const sub = walk(n.children ?? [])
          counts.set(n.path, sub)
          files += sub
        } else {
          files++
        }
      }
      return files
    }
    walk(nodes ?? [])
    return { allDirs: dirs, fileCounts: counts }
  }, [nodes])

  if (error) {
    return <p className="px-1 text-sm text-red-600 dark:text-red-400">Could not load the wiki tree</p>
  }
  if (nodes === null) {
    return <p className="px-1 text-sm text-subtle">Loading…</p>
  }
  if (nodes.length === 0) {
    return <p className="px-1 text-sm text-subtle">No notes yet — your wiki is empty.</p>
  }

  return (
    <div>
      <div className="mb-1 flex items-center justify-end gap-1">
        <button
          type="button"
          onClick={() => setExpandedKeys(new Set(allDirs))}
          aria-label="Expand all folders"
          title="Expand all"
          className="rounded p-1 text-subtle transition hover:bg-raised hover:text-ink"
        >
          <ChevronsUpDown className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={() => setExpandedKeys(new Set())}
          aria-label="Collapse all folders"
          title="Collapse all"
          className="rounded p-1 text-subtle transition hover:bg-raised hover:text-ink"
        >
          <ChevronsDownUp className="h-4 w-4" />
        </button>
      </div>
      <Tree<TreeNode>
        nodes={nodes}
        getKey={(n) => n.path}
        getLabel={(n) => n.name}
        isDir={(n) => n.is_dir}
        getChildren={(n) => n.children ?? []}
        renderIcon={(n, expanded) =>
          n.is_dir
            ? expanded
              ? <FolderOpen className="h-4 w-4" />
              : <Folder className="h-4 w-4" />
            : <FileText className="h-4 w-4" />}
        renderTrailing={(n) =>
          n.is_dir ? (
            <span className="shrink-0 rounded-full bg-raised px-1.5 text-[10px] text-subtle">
              {fileCounts.get(n.path) ?? 0}
            </span>
          ) : null}
        onSelect={(n) => { if (!n.is_dir) onOpenNote(n.path) }}
        selectedKey={openPath}
        expandedKeys={expandedKeys}
        onExpandedChange={setExpandedKeys}
      />
    </div>
  )
}
