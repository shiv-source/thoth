import { Card, List } from 'antd'
import { FolderOpenOutlined } from '@ant-design/icons'

// InboxCard is the Overview "Inbox needs attention" widget: the waiting-capture
// count and the inbox/ file names it refers to.
export function InboxCard({ count, files }: { count: number; files: string[] }) {
    return (
        <Card size="small" title="Inbox needs attention">
            <p className="flex items-center gap-2 text-sm text-ink">
                <FolderOpenOutlined className="h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                {count} capture{count === 1 ? '' : 's'} waiting
            </p>
            <List
                size="small"
                dataSource={files}
                renderItem={(f) => <List.Item className="truncate text-xs text-subtle">inbox/{f}</List.Item>}
            />
            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
        </Card>
    )
}
