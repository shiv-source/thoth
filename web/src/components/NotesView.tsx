import { useMemo, useState } from 'react'
import { FileText } from 'lucide-react'
import { WikiTree } from './WikiTree'

// NotesView is the browse-and-read surface: the wiki tree on the left, the
// note content on the right. The reader pane currently shows a placeholder —
// the inline viewer (replacing the floating NoteViewer overlay) lands in the
// wiring pass after the shell review.
export function NotesView({ openPath, onOpenNote }: { openPath: string | null; onOpenNote: (path: string) => void }) {
    const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())
    const onDirsChange = useMemo(() => (_dirs: Set<string>) => {}, [])

    return (
        <div className="flex min-h-0 flex-1">
            <aside className="flex w-72 shrink-0 flex-col border-r border-line bg-surface">
                <header className="border-b border-line px-4 py-3">
                    <h1 className="text-sm font-medium text-ink">Wiki</h1>
                </header>
                <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
                    <WikiTree
                        openPath={openPath}
                        onOpenNote={onOpenNote}
                        expandedKeys={expandedKeys}
                        onExpandedChange={setExpandedKeys}
                        onDirsChange={onDirsChange}
                    />
                </div>
            </aside>
            <main className="flex min-w-0 flex-1 flex-col items-center justify-center gap-3 text-subtle">
                <span className="text-3xl" aria-hidden="true">
                    🦉
                </span>
                <p className="text-sm">Select a note to read it here</p>
                <p className="max-w-sm text-center text-xs">
                    <FileText className="mr-1 inline h-3.5 w-3.5" aria-hidden="true" />
                    Inline reader + backlinks land in the wiring pass.
                </p>
            </main>
        </div>
    )
}
