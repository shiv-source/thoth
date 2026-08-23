import { Card, Listy } from 'antd'
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
            <Listy
                items={files}
                rowKey={(f) => f}
                className="divide-y divide-line"
                classNames={{ item: 'p-0!' }}
                itemRender={(f) => <div className="truncate py-1 text-xs text-subtle">inbox/{f}</div>}
            />
            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
        </Card>
    )
}
