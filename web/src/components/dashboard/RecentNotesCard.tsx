import { Card, Listy } from 'antd'

// RecentNotesCard is the Overview "Recent notes" widget: the wiki paths of
// the most recently touched notes.
export function RecentNotesCard({ notes }: { notes: string[] }) {
    return (
        <Card size="small" title="Recent notes">
            <Listy
                items={notes}
                rowKey={(p) => p}
                className="divide-y divide-line"
                classNames={{ item: 'p-0!' }}
                itemRender={(p) => <div className="py-1 text-sm text-ink">{p}</div>}
            />
            <p className="mt-3 text-xs text-subtle">mock data — wire the index's updated_at</p>
        </Card>
    )
}
