import type { ComponentType } from 'react'
import { Card } from 'antd'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'

// StatTile is a KPI tile: a tinted icon chip on the left with the label and
// a large value beside it, plus an optional trend delta (a leading '+' tints
// the delta success).
export function StatTile({
    icon: Icon,
    label,
    value,
    delta
}: {
    icon: ComponentType<AntdIconProps>
    label: string
    value: string
    delta?: string
}) {
    const positive = delta?.startsWith('+')
    return (
        <Card size="small">
            <div className="flex items-start gap-3">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-accent-soft text-accent">
                    <Icon aria-hidden="true" />
                </span>
                <div className="min-w-0 flex-1">
                    <div className="truncate text-xs font-medium text-faint">{label}</div>
                    <div className="mt-0.5 flex items-baseline gap-1.5">
                        <span className="text-2xl font-semibold leading-none text-heading">{value}</span>
                        {delta && (
                            <span className={`shrink-0 text-xs ${positive ? 'text-success' : 'text-subtle'}`}>
                                {delta}
                            </span>
                        )}
                    </div>
                </div>
            </div>
        </Card>
    )
}
