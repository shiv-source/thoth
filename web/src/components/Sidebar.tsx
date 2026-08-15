import { SearchPanel } from './SearchPanel'
import { WikiTree } from './WikiTree'

export function Sidebar({ openPath, onOpenNote }: {
  openPath: string | null
  onOpenNote: (path: string) => void
}) {
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
        <WikiTree openPath={openPath} onOpenNote={onOpenNote} />
      </div>
    </aside>
  )
}
