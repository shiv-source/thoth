import { Card, List } from 'antd'

// RecentNotesCard is the Overview "Recent notes" widget: the wiki paths of
// the most recently touched notes.
export function RecentNotesCard({ notes }: { notes: string[] }) {
    return (
        <Card size="small" title="Recent notes">
            <List
                size="small"
                dataSource={notes}
                renderItem={(p) => <List.Item className="text-sm text-ink">{p}</List.Item>}
            />
            <p className="mt-3 text-xs text-subtle">mock data — wire the index's updated_at</p>
        </Card>
    )
}
