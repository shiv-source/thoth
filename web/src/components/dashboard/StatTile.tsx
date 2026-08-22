import type { ComponentType } from 'react'
import { Card, Statistic } from 'antd'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'

// StatTile is a KPI tile: an antd Card with a Statistic whose prefix icon
// carries the accent.
export function StatTile({
    icon: Icon,
    label,
    value
}: {
    icon: ComponentType<AntdIconProps>
    label: string
    value: string
}) {
    return (
        <Card size="small">
            <Statistic title={label} value={value} prefix={<Icon aria-hidden="true" className="mr-1 text-accent" />} />
        </Card>
    )
}
