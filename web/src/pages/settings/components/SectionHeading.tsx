import type { ComponentType, ReactNode } from 'react'
import { Flex } from 'antd'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'

// SectionHeading is the icon'd micro-title that opens each card section
// (the DashboardPage kicker pattern, inside a Card).
export function SectionHeading({ icon: Icon, children }: { icon: ComponentType<AntdIconProps>; children: ReactNode }) {
    return (
        <Flex align="center" gap={8} className="mb-1!">
            <Icon aria-hidden="true" className="text-subtle" />
            <h3 className="text-xs font-medium uppercase tracking-wide text-subtle">{children}</h3>
        </Flex>
    )
}
