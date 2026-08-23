import { useEffect, useMemo } from 'react'
import { Button, Tooltip } from 'antd'
import { CompressOutlined, ExpandOutlined, FileTextOutlined } from '@ant-design/icons'
import { collectTreeInfo, selectNotesExpandedKeys, selectWikiNodes, setNotesExpandedKeys } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { AppHeader } from '../../shared/AppHeader'
import { EmptyState } from '../../shared/EmptyState'
import { NoteViewer } from '../../components/notes/NoteViewer'
import { WikiTree } from '../../components/notes/WikiTree'

// NotesPage is the browse-and-read surface: the wiki tree on the left (with
// the expand/collapse-all toggle), and the note reader as a Drawer on the
// right. Tree expansion lives in the ui slice.
export function NotesPage({
    openPath,
    onOpenNote,
    onOpenSettings
}: {
    openPath: string | null
    onOpenNote: (path: string | null) => void
    onOpenSettings: () => void
}) {
    const dispatch = useAppDispatch()
    const nodes = useAppSelector(selectWikiNodes)
    const expandedKeys = useAppSelector(selectNotesExpandedKeys)
    const { allDirs } = useMemo(() => collectTreeInfo(nodes ?? []), [nodes])
    const allExpanded = allDirs.size > 0 && expandedKeys.length >= allDirs.size

    // The open note's ancestor folders stay expanded, so a reload (or any
    // navigation into a note) shows where the note lives in the tree.
    useEffect(() => {
        if (!openPath) return
        const dirs: string[] = []
        let idx = openPath.indexOf('/')
        while (idx !== -1) {
            dirs.push(openPath.slice(0, idx))
            idx = openPath.indexOf('/', idx + 1)
        }
        dispatch(setNotesExpandedKeys([...new Set([...expandedKeys, ...dirs])]))
    }, [openPath, dispatch])

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <AppHeader title="Wiki" onOpenSettings={onOpenSettings} />
            <div className="flex min-h-0 flex-1">
                <aside className="flex w-72 shrink-0 flex-col border-r border-line bg-surface">
                    <header className="flex shrink-0 items-center justify-between border-b border-line px-4 py-2.5">
                        <h1 className="text-sm font-medium text-ink">Folders</h1>
                        <Tooltip title={allExpanded ? 'Collapse all' : 'Expand all'}>
                            <Button
                                type="text"
                                size="small"
                                aria-label={allExpanded ? 'Collapse all folders' : 'Expand all folders'}
                                icon={
                                    allExpanded ? (
                                        <CompressOutlined aria-hidden="true" />
                                    ) : (
                                        <ExpandOutlined aria-hidden="true" />
                                    )
                                }
                                onClick={() => dispatch(setNotesExpandedKeys(allExpanded ? [] : [...allDirs]))}
                            />
                        </Tooltip>
                    </header>
                    <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
                        <WikiTree openPath={openPath} onOpenNote={onOpenNote} />
                    </div>
                </aside>
                <main className="flex min-w-0 flex-1 flex-col">
                    {openPath ? (
                        <NoteViewer path={openPath} onClose={() => onOpenNote(null)} />
                    ) : (
                        <div className="m-auto">
                            <EmptyState
                                icon={<FileTextOutlined aria-hidden="true" />}
                                title="Select a note to read it here"
                                description="Pick a note in the tree to preview its contents."
                            />
                        </div>
                    )}
                </main>
            </div>
        </div>
    )
}
