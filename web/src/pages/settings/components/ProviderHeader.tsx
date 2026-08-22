import { Flex, Tag } from 'antd'
import { GlobalOutlined } from '@ant-design/icons'

// ProviderHeader is a Collapse panel title: the provider name plus status
// chips (model count, key state, endpoint). The key/endpoint states read as
// colored dots so the overall health of the provider is scannable at a
// glance, with the counts spelled out next to them.
export function ProviderHeader({
    provider,
    modelCount,
    hasKey,
    baseURL
}: {
    provider: string
    modelCount: number
    hasKey: boolean
    baseURL: string
}) {
    return (
        <Flex align="center" justify="space-between" gap={8} className="pr-2">
            <Flex align="center" gap={8} className="min-w-0">
                <span
                    className={`h-2 w-2 shrink-0 rounded-full ${hasKey ? 'bg-accent' : 'bg-line'}`}
                    aria-hidden="true"
                />
                <span className="truncate font-medium text-heading">{provider === '' ? 'Unassigned' : provider}</span>
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
            </Flex>
        </Flex>
    )
}
