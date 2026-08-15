import { useCallback, useEffect, useMemo, useState } from 'react'
import { FileText, Folder, FolderOpen } from 'lucide-react'
import { api, type TreeNode } from '../api/client'
import { Tree } from './Tree'

// WikiTree renders the wiki directory as a folder tree via the reusable
// Tree component: folders collapse with chevrons, files open the note.
// Expansion is controlled by the caller (the sidebar header owns the
// expand/collapse-all toggle); onDirsChange reports the dir keys so the
// caller can expand everything.
export function WikiTree({
    openPath,
    onOpenNote,
    expandedKeys,
    onExpandedChange,
    onDirsChange
}: {
    openPath: string | null
    onOpenNote: (path: string) => void
    expandedKeys: Set<string>
    onExpandedChange: (next: Set<string>) => void
    onDirsChange?: (dirs: Set<string>) => void
}) {
    const [nodes, setNodes] = useState<TreeNode[] | null>(null)
    const [error, setError] = useState(false)

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

    useEffect(() => {
        if (nodes !== null) onDirsChange?.(allDirs)
    }, [allDirs, nodes, onDirsChange])

    // Stable accessors are the contract Tree's memoized rows rely on: without
    // them every WikiTree render (e.g. each openNote change) re-renders every
    // visible row. Declared before the early returns — hooks must run on every
    // render regardless of state.
    const getKey = useCallback((n: TreeNode) => n.path, [])
    const getLabel = useCallback((n: TreeNode) => n.name, [])
    const isDir = useCallback((n: TreeNode) => n.is_dir, [])
    const getChildren = useCallback((n: TreeNode) => n.children ?? [], [])
    const renderIcon = useCallback(
        (n: TreeNode, expanded: boolean) =>
            n.is_dir ? (
                expanded ? (
                    <FolderOpen className="h-4 w-4" />
                ) : (
                    <Folder className="h-4 w-4" />
                )
            ) : (
                <FileText className="h-4 w-4" />
            ),
        []
    )
    const renderTooltip = useCallback(
        (n: TreeNode) =>
            n.is_dir
                ? `${fileCounts.get(n.path) ?? 0} file${(fileCounts.get(n.path) ?? 0) === 1 ? '' : 's'}`
                : undefined,
        [fileCounts]
    )
    const onSelect = useCallback(
        (n: TreeNode) => {
            if (!n.is_dir) onOpenNote(n.path)
        },
        [onOpenNote]
    )

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
            <Tree<TreeNode>
                nodes={nodes}
                getKey={getKey}
                getLabel={getLabel}
                isDir={isDir}
                getChildren={getChildren}
                renderIcon={renderIcon}
                renderTooltip={renderTooltip}
                onSelect={onSelect}
                selectedKey={openPath}
                expandedKeys={expandedKeys}
                onExpandedChange={onExpandedChange}
            />
        </div>
    )
}
