import { Button, Listy } from 'antd'
import { ClockCircleOutlined } from '@ant-design/icons'

// RecentSearches is the empty-query state of SearchPanel: the persisted
// recent-search list with a Clear action. Picking one re-runs the query.
export function RecentSearches({
    history,
    onPick,
    onClear
}: {
    history: string[]
    onPick: (query: string) => void
    onClear: () => void
}) {
    if (history.length === 0) return null
    return (
        <div className="mt-1.5">
            <div className="flex items-center justify-between px-1">
                <p className="text-xs font-medium uppercase tracking-wide text-subtle">Recent searches</p>
                <Button type="link" size="small" onClick={onClear}>
                    Clear
                </Button>
            </div>
            <Listy
                items={history}
                rowKey={(h) => h}
                classNames={{ item: 'p-0!' }}
                itemRender={(h) => (
                    <Button
                        type="text"
                        block
                        className="flex h-auto items-center justify-start py-1.5 text-left"
                        icon={<ClockCircleOutlined aria-hidden="true" />}
                        onClick={() => onPick(h)}
                    >
                        <span className="truncate">{h}</span>
                    </Button>
                )}
            />
        </div>
    )
}
