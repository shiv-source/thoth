import { useState } from 'react'
import { ChevronsDownUp, ChevronsUpDown } from 'lucide-react'
import { NoteViewer } from './NoteViewer'
import { Tooltip } from './Tooltip'
import { TopBar } from './TopBar'
import { WikiTree } from './WikiTree'

// NotesView is the browse-and-read surface: the wiki tree on the left (with
// the expand/collapse-all toggle that used to live in the sidebar), and the
// note content rendered inline on the right — no drawer.
export function NotesView({
    openPath,
    onOpenNote,
    onOpenSettings
}: {
    openPath: string | null
    onOpenNote: (path: string | null) => void
    onOpenSettings: () => void
}) {
    const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())
    const [allDirs, setAllDirs] = useState<Set<string>>(() => new Set())
    const allExpanded = allDirs.size > 0 && expandedKeys.size >= allDirs.size

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <TopBar title="Wiki" onOpenSettings={onOpenSettings} />
            <div className="flex min-h-0 flex-1">
                <aside className="flex w-72 shrink-0 flex-col border-r border-line bg-surface">
                    <header className="flex shrink-0 items-center justify-between border-b border-line px-4 py-2.5">
                        <h1 className="text-sm font-medium text-ink">Folders</h1>
                        <Tooltip label={allExpanded ? 'Collapse all' : 'Expand all'}>
                            <button
                                type="button"
                                onClick={() => setExpandedKeys(allExpanded ? new Set() : new Set(allDirs))}
                                aria-label={allExpanded ? 'Collapse all folders' : 'Expand all folders'}
                                className="rounded p-1 text-subtle transition hover:bg-raised hover:text-ink"
                            >
                                {allExpanded ? (
                                    <ChevronsDownUp className="h-4 w-4" />
                                ) : (
                                    <ChevronsUpDown className="h-4 w-4" />
                                )}
                            </button>
                        </Tooltip>
                    </header>
                    <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
                        <WikiTree
                            openPath={openPath}
                            onOpenNote={(p) => onOpenNote(p)}
                            expandedKeys={expandedKeys}
                            onExpandedChange={setExpandedKeys}
                            onDirsChange={setAllDirs}
                        />
                    </div>
                </aside>
                <main className="flex min-w-0 flex-1 flex-col">
                    {openPath ? (
                        <NoteViewer path={openPath} onClose={() => onOpenNote(null)} />
                    ) : (
                        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-subtle">
                            <span className="text-3xl" aria-hidden="true">
                                🦉
                            </span>
                            <p className="text-sm">Select a note to read it here</p>
                            <p className="max-w-sm text-center text-xs">Backlinks land in the wiring pass.</p>
                        </div>
                    )}
                </main>
            </div>
        </div>
    )
}
