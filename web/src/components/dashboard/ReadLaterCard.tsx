import { useCallback, useEffect, useState } from 'react'
import { App, Button, Card, Empty, Flex, Space, Typography } from 'antd'
import { CheckOutlined } from '@ant-design/icons'
import { api, isConflict, type ReadLaterItem } from '../../api/client'

// ReadLaterCard is the Overview "Read later" widget: the links/read-later.md
// queue, triageable right from the dashboard. Each queued link can be opened,
// promoted to a bookmark (adding it to links/bookmarks.md and clearing the
// queue), or marked done (removed from the queue).
export function ReadLaterCard() {
    const { message } = App.useApp()
    const [items, setItems] = useState<ReadLaterItem[]>([])
    const [loading, setLoading] = useState(false)

    const refresh = useCallback(async () => {
        setLoading(true)
        try {
            const res = await api.readLater()
            setItems(res.items)
        } catch {
            setItems([])
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        void refresh()
    }, [refresh])

    const bookmark = async (item: ReadLaterItem) => {
        try {
            await api.capture({ kind: 'bookmark', url: item.url, title: item.title, reason: item.reason })
            message.success('Added to bookmarks')
        } catch (err) {
            if (isConflict(err)) {
                message.info('Already in bookmarks')
            } else {
                message.error('Could not bookmark')
                return // keep it queued on a network failure
            }
        }
        await api.removeReadLater(item.url)
        void refresh()
    }

    const done = async (item: ReadLaterItem) => {
        await api.removeReadLater(item.url)
        message.success('Removed from read later')
        void refresh()
    }

    return (
        <Card size="small" title="Read later" loading={loading}>
            {items.length === 0 ? (
                <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={<span className="text-xs text-subtle">Nothing queued</span>}
                />
            ) : (
                <Flex vertical gap={10}>
                    {items.map((item) => (
                        <Flex key={item.url} align="center" justify="space-between" gap={8}>
                            <Flex vertical style={{ minWidth: 0 }} gap={0}>
                                <Typography.Text strong ellipsis>
                                    <a href={item.url} target="_blank" rel="noopener noreferrer">
                                        {item.title}
                                    </a>
                                </Typography.Text>
                                {item.reason && (
                                    <Typography.Text type="secondary" style={{ fontSize: 12 }} ellipsis>
                                        {item.reason}
                                    </Typography.Text>
                                )}
                            </Flex>
                            <Space size={4}>
                                <Button size="small" onClick={() => void bookmark(item)}>
                                    Bookmark
                                </Button>
                                <Button
                                    size="small"
                                    aria-label={`Done ${item.title}`}
                                    icon={<CheckOutlined aria-hidden="true" />}
                                    onClick={() => void done(item)}
                                />
                            </Space>
                        </Flex>
                    ))}
                </Flex>
            )}
        </Card>
    )
}
