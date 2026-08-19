import { useCallback, useEffect, useMemo, useRef } from 'react'
import { Tooltip, Tree } from 'antd'
import type { DataNode } from 'antd/es/tree'
import { FileTextOutlined, FolderOpenOutlined, FolderOutlined, WarningOutlined } from '@ant-design/icons'
import type { TreeNode } from '../api/client'
import {
    collectTreeInfo,
    fetchTree,
    selectConnectionStatus,
    selectNotesExpandedKeys,
    selectWikiError,
    selectWikiLoading,
    selectWikiNodes,
    setNotesExpandedKeys
} from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import type { ConnectionStatus } from '../ws/chat'

// WikiDataNode carries the per-directory file count alongside antd's own
// fields so the title renderer can show the hover tooltip.
interface WikiDataNode extends DataNode {
    fileCount?: number
    error?: string
}

// toTreeData maps the API nodes onto antd's shape, tagging directories
// with their recursive file counts.
function toTreeData(nodes: TreeNode[], counts: Map<string, number>): WikiDataNode[] {
    return nodes.map((n) => ({
        key: n.path,
        title: n.name,
        isLeaf: !n.is_dir,
        children: n.is_dir ? toTreeData(n.children ?? [], counts) : undefined,
        fileCount: n.is_dir ? counts.get(n.path) : undefined,
        error: n.error
    }))
}

// WikiTree renders the wiki directory through antd's DirectoryTree: folder
// clicks expand/collapse, file clicks open the note. The tree data and the
// expansion state both live in Redux (wikiSlice + uiSlice), so the tree
// survives view switches and the expand-all toggle in NotesView works
// without prop drilling.
export function WikiTree({ openPath, onOpenNote }: { openPath: string | null; onOpenNote: (path: string) => void }) {
    const dispatch = useAppDispatch()
    const nodes = useAppSelector(selectWikiNodes)
    const loading = useAppSelector(selectWikiLoading)
    const error = useAppSelector(selectWikiError)
    const status = useAppSelector(selectConnectionStatus)
    const expandedKeys = useAppSelector(selectNotesExpandedKeys)
    const prevStatus = useRef<ConnectionStatus | null>(null)

    // Fetch once per connection: the initial 'connected' state covers mount,
    // and a reconnect edge reseeds the tree because wiki_changed frames may
    // have been missed while the socket was down. Per-change refetches ride
    // the wiki_changed frames (useChat), so no polling is needed here.
    useEffect(() => {
        if (prevStatus.current !== 'connected' && status === 'connected') void dispatch(fetchTree())
        prevStatus.current = status
    }, [status, dispatch])

    // Notes written outside the app (terminal Claude, vim) while no socket is
    // connected show up when the window regains focus.
    useEffect(() => {
        const load = () => void dispatch(fetchTree())
        window.addEventListener('focus', load)
        return () => window.removeEventListener('focus', load)
    }, [dispatch])

    const { fileCounts } = useMemo(() => collectTreeInfo(nodes ?? []), [nodes])
    const treeData = useMemo(() => toTreeData(nodes ?? [], fileCounts), [nodes, fileCounts])

    // One icon per row: directories replace the default caret with an
    // open/closed folder (switcherIcon), files show a document icon in the
    // title. Directories also carry a hover tooltip with their recursive
    // file count.
    const renderTitle = useCallback((node: WikiDataNode) => {
        const title = node.title as string
        if (node.isLeaf) {
            return (
                <span className="inline-flex items-center gap-1.5">
                    <FileTextOutlined aria-hidden="true" className="text-subtle" />
                    <span>{title}</span>
                </span>
            )
        }
        // An unreadable directory keeps its folder node with a warning so
        // the rest of the tree still renders (see internal/wiki tree()).
        if (node.error) {
            return (
                <Tooltip title={node.error}>
                    <span className="inline-flex items-center gap-1.5">
                        <WarningOutlined aria-hidden="true" className="text-amber-500" />
                        <span>{title}</span>
                    </span>
                </Tooltip>
            )
        }
        const count = node.fileCount ?? 0
        return <Tooltip title={`${count} file${count === 1 ? '' : 's'}`}>{title}</Tooltip>
    }, [])

    const renderSwitcher = useCallback((props: { expanded?: boolean }) => {
        return props.expanded ? (
            <FolderOpenOutlined aria-hidden="true" className="text-subtle" />
        ) : (
            <FolderOutlined aria-hidden="true" className="text-subtle" />
        )
    }, [])

    if (error) {
        return <p className="px-1 text-sm text-red-600">Could not load the wiki tree</p>
    }
    if (nodes === null || loading) {
        return <p className="px-1 text-sm text-subtle">Loading…</p>
    }
    if (nodes.length === 0) {
        return <p className="px-1 text-sm text-subtle">No notes yet — your wiki is empty.</p>
    }

    return (
        <Tree.DirectoryTree
            treeData={treeData}
            titleRender={renderTitle}
            switcherIcon={renderSwitcher}
            // DirectoryTree defaults showIcon on and would render its own
            // folder glyph next to the custom switcher — one icon per row
            // means turning its icon slot off (files carry theirs in the
            // title renderer).
            showIcon={false}
            expandedKeys={expandedKeys}
            onExpand={(keys) => dispatch(setNotesExpandedKeys(keys as string[]))}
            selectedKeys={openPath ? [openPath] : []}
            onSelect={(_keys, { node }) => {
                if (node.isLeaf) onOpenNote(node.key as string)
            }}
            // A local wiki tree is small — virtualization buys nothing and
            // its height measurement does not work under jsdom. Motion is
            // off for the same reason (its leave callbacks never complete
            // under jsdom) and because instant expand/collapse feels
            // snappier for a local file tree.
            virtual={false}
            motion={false}
        />
    )
}
