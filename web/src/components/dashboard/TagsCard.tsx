import { Button, Card } from 'antd'
import { TagOutlined } from '@ant-design/icons'

// TagsCard is the Overview "Tags" widget: the wiki's tags as round chips that
// jump to the search view.
export function TagsCard({ tags, onOpen }: { tags: string[]; onOpen: () => void }) {
    return (
        <Card size="small" title="Tags">
            <div className="flex flex-wrap gap-2">
                {tags.map((t) => (
                    <Button
                        key={t}
                        shape="round"
                        size="small"
                        icon={<TagOutlined aria-hidden="true" />}
                        onClick={onOpen}
                    >
                        #{t}
                    </Button>
                ))}
            </div>
            <p className="mt-3 text-xs text-subtle">mock data — index tags</p>
        </Card>
    )
}
