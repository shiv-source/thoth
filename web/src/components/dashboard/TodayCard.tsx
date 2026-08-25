import { Badge, Card, Tag, Timeline } from 'antd'
import { CalendarOutlined, InboxOutlined, RightOutlined } from '@ant-design/icons'
import { EmptyState } from '../../shared/EmptyState'

// TodayEvent is one "Today" timeline entry: a clock time, the title, an
// optional wiki path, and whether it is a meeting or a capture (the two
// choose the timeline dot color and kind tag).
export interface TodayEvent {
    id: string
    time: string
    title: string
    path?: string
    kind: 'meeting' | 'capture'
}

// TodayCard is the Overview "Today" widget: meetings and captures on the
// day's timeline, each opening its note when it has one.
export function TodayCard({ events, onOpen }: { events: TodayEvent[]; onOpen: (path: string) => void }) {
    if (events.length === 0) {
        return (
            <Card size="small" title="Today">
                <EmptyState
                    icon={<CalendarOutlined className="h-5 w-5" aria-hidden="true" />}
                    title="Nothing scheduled"
                    description="Your meetings and captures for today will show up here."
                />
            </Card>
        )
    }

    return (
        <Card
            size="small"
            title="Today"
            extra={<Badge count={events.length} color="var(--ant-color-fill-secondary)" />}
            styles={{ body: { padding: '4px 16px 12px' } }}
        >
            <Timeline
                items={events.map((e) => ({
                    key: e.id,
                    title: <span className="font-mono text-xs text-subtle">{e.time}</span>,
                    color: e.kind === 'meeting' ? 'blue' : 'green',
                    content: (
                        <button
                            type="button"
                            onClick={() => onOpen(e.path ?? e.id)}
                            className="group flex w-full items-center gap-2 rounded-md py-2 pr-2 text-left transition-colors hover:bg-raised"
                        >
                            <span className="min-w-0 flex-1">
                                <span className="flex items-center gap-1.5">
                                    <span className="min-w-0 flex-1 truncate text-sm font-medium text-ink">
                                        {e.title}
                                    </span>
                                    <Tag
                                        variant="filled"
                                        icon={e.kind === 'meeting' ? <CalendarOutlined /> : <InboxOutlined />}
                                        color={e.kind === 'meeting' ? 'blue' : 'green'}
                                        className="m-0! shrink-0"
                                    >
                                        {e.kind}
                                    </Tag>
                                </span>
                                {e.path && <span className="block truncate text-xs text-subtle">{e.path}</span>}
                            </span>
                            <RightOutlined
                                className="h-3.5 w-3.5 shrink-0 text-subtle opacity-0 transition group-hover:opacity-100"
                                aria-hidden="true"
                            />
                        </button>
                    )
                }))}
            />
        </Card>
    )
}
