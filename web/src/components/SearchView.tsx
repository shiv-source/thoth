import { SearchPanel } from './SearchPanel'
import { AppHeader } from './AppHeader'

// SearchView is the full-page search surface — the same search that lives in
// the sidebar, with room for richer results (kind filters, previews) later.
export function SearchView({
    onOpenNote,
    onOpenSettings
}: {
    onOpenNote: (path: string) => void
    onOpenSettings: () => void
}) {
    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <AppHeader title="Search" onOpenSettings={onOpenSettings} />
            <div className="mx-auto w-full max-w-2xl px-4 py-5">
                <SearchPanel onOpen={onOpenNote} />
            </div>
        </div>
    )
}
