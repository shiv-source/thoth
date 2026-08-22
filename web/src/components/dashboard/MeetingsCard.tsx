import { Card, List } from 'antd'
import { ClockCircleOutlined } from '@ant-design/icons'

// Meeting is one row of today's meetings widget.
export interface Meeting {
    time: string
    title: string
    path: string
}

// MeetingsCard is the Overview "Today's meetings" widget: each meeting as a
// hover-reveal time chip + title/path row.
export function MeetingsCard({ meetings }: { meetings: Meeting[] }) {
    return (
        <Card size="small" title="Today's meetings">
            <List
                size="small"
                dataSource={meetings}
                renderItem={(m) => (
                    <List.Item className="group px-2 hover:bg-raised">
                        <span className="mr-2.5 shrink-0 rounded-md bg-raised px-1.5 py-0.5 font-mono text-xs text-subtle">
                            {m.time}
                        </span>
                        <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm text-ink">{m.title}</span>
                            <span className="block truncate text-xs text-subtle">{m.path}</span>
                        </span>
                        <ClockCircleOutlined
                            className="h-4 w-4 shrink-0 text-subtle opacity-0 transition group-hover:opacity-100"
                            aria-hidden="true"
                        />
                    </List.Item>
                )}
            />
            <p className="mt-3 text-xs text-subtle">mock data — index by kind</p>
        </Card>
    )
}
