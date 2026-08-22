import { Empty, List } from 'antd'
import type { SearchResult } from '../../api/client'

// SearchResults renders the current query's matches: a loading line, an
// Empty state, or the result list. Keyboard/mouse highlights come in as the
// active index; picking or hovering routes through onOpen/onHover.
export function SearchResults({
    results,
    loading,
    active,
    onOpen,
    onHover
}: {
    results: SearchResult[]
    loading: boolean
    active: number
    onOpen: (path: string) => void
    onHover: (index: number) => void
}) {
    return (
        <div className="mt-1.5">
            {loading && <p className="px-1 text-xs text-subtle">Searching…</p>}
            {!loading && results.length === 0 && (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No notes match." className="py-4" />
            )}
            {results.length > 0 && (
                <List
                    size="small"
                    dataSource={results}
                    renderItem={(r, i) => (
                        <List.Item
                            onClick={() => onOpen(r.path)}
                            onMouseEnter={() => onHover(i)}
                            className={`cursor-pointer rounded-lg px-2 ${i === active ? 'bg-accent-soft' : 'hover:bg-raised'}`}
                        >
                            <List.Item.Meta
                                title={<span className="text-sm font-medium">{r.title}</span>}
                                description={
                                    <>
                                        <div className="truncate text-xs text-subtle">{r.path}</div>
                                        {/* Safe: the server escapes note text before building the
                                            snippet; only its <mark> markers survive as real tags. */}
                                        <div
                                            className="mt-0.5 line-clamp-2 text-xs"
                                            dangerouslySetInnerHTML={{ __html: r.snippet }}
                                        />
                                    </>
                                }
                            />
                        </List.Item>
                    )}
                />
            )}
        </div>
    )
}
