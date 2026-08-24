import { Button, Flex, Popconfirm, Tag } from 'antd'
import { DeleteOutlined, EditOutlined, GlobalOutlined } from '@ant-design/icons'

// ProviderHeader is a Collapse panel title: the provider name plus status
// chips (model count, key state, endpoint) and the edit/delete actions. The
// key/endpoint states read as colored dots so the overall health of the
// provider is scannable at a glance, with the counts spelled out next to them.
export function ProviderHeader({
    name,
    modelCount,
    hasKey,
    baseURL,
    hasCustomHeaders = false,
    onEdit,
    onDelete
}: {
    name: string
    modelCount: number
    hasKey: boolean
    baseURL: string
    hasCustomHeaders?: boolean
    onEdit?: () => void
    onDelete?: () => void
}) {
    return (
        <div className="flex items-center justify-between gap-2 pr-2">
            <Flex align="center" gap={8} className="min-w-0">
                <span
                    className={`h-2 w-2 shrink-0 rounded-full ${hasKey ? 'bg-accent' : 'bg-line'}`}
                    aria-hidden="true"
                />
                <span className="truncate font-medium text-heading">{name}</span>
            </Flex>
            <Flex align="center" gap={6} className="shrink-0">
                <Tag>
                    {modelCount} model{modelCount === 1 ? '' : 's'}
                </Tag>
                <Tag color={hasKey ? 'success' : 'default'}>{hasKey ? 'key set' : 'no key'}</Tag>
                <Tag
                    icon={baseURL !== '' ? <GlobalOutlined aria-hidden="true" /> : undefined}
                    color={baseURL !== '' ? 'blue' : 'default'}
                >
                    {baseURL !== '' ? 'custom endpoint' : 'default endpoint'}
                </Tag>
                {hasCustomHeaders && <Tag color="purple">custom headers</Tag>}
                {onEdit && onDelete && (
                    <Flex align="center" gap={0} onClick={(e) => e.stopPropagation()}>
                        <Button
                            size="small"
                            type="text"
                            aria-label={`Edit ${name}`}
                            icon={<EditOutlined aria-hidden="true" />}
                            onClick={onEdit}
                        />
                        <Popconfirm
                            title="Delete this provider?"
                            description="Its models are removed too."
                            trigger="click"
                            onConfirm={onDelete}
                        >
                            <Button
                                size="small"
                                type="text"
                                danger
                                aria-label={`Delete ${name}`}
                                icon={<DeleteOutlined aria-hidden="true" />}
                            />
                        </Popconfirm>
                    </Flex>
                )}
            </Flex>
        </div>
    )
}
