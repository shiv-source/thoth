import { Badge, Button, Card, Tag } from 'antd'
import { TagOutlined } from '@ant-design/icons'

// TagsCard is the Overview "Tags" widget: the wiki's tags as antd Tag chips
// that jump to the search view.
export function TagsCard({ tags, onOpen }: { tags: string[]; onOpen: () => void }) {
    return (
        <Card size="small" title="Tags" extra={<Badge count={tags.length} color="var(--ant-color-fill-secondary)" />}>
            <div className="flex flex-wrap gap-1.5">
                {tags.map((t) => (
                    <Button
                        key={t}
                        type="text"
                        size="small"
                        onClick={onOpen}
                        aria-label={`#${t}`}
                        className="h-auto! px-1! py-0.5!"
                    >
                        <Tag variant="filled" icon={<TagOutlined aria-hidden="true" />} className="m-0!">
                            {t}
                        </Tag>
                    </Button>
                ))}
            </div>
            <p className="mt-3 text-xs text-subtle">mock data — index tags</p>
        </Card>
    )
}
