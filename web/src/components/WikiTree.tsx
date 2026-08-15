import { useEffect, useState } from 'react'
import { FileText, Folder, FolderOpen } from 'lucide-react'
import { api, type TreeNode } from '../api/client'
import { Tree } from './Tree'

// WikiTree renders the wiki directory as a folder tree via the reusable
// Tree component: folders collapse with chevrons, files open the note.
export function WikiTree({ openPath, onOpenNote }: { openPath: string | null; onOpenNote: (path: string) => void }) {
  const [nodes, setNodes] = useState<TreeNode[] | null>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    api.tree()
      .then((r) => setNodes(r.nodes))
      .catch(() => setError(true))
  }, [])

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
      onSelect={(n) => { if (!n.is_dir) onOpenNote(n.path) }}
      selectedKey={openPath}
    />
  )
}
