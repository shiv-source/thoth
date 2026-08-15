import { SearchPanel } from './SearchPanel'

// SearchView is the full-page search surface — the same search that lives in
// the sidebar, with room for richer results (kind filters, previews) later.
export function SearchView({ onOpenNote }: { onOpenNote: (path: string) => void }) {
    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <header className="border-b border-line bg-surface px-4 py-3">
                <h1 className="text-sm font-medium text-ink">Search</h1>
            </header>
            <div className="mx-auto w-full max-w-2xl px-4 py-5">
                <SearchPanel onOpen={onOpenNote} />
            </div>
        </div>
    )
}
