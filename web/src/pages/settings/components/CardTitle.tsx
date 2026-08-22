import type { ComponentType, ReactNode } from 'react'
import { Flex } from 'antd'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'

// CardTitle is the icon'd Card title shared by every settings section, so
// the four pages read as one system (General / Providers / GitHub / Checks).
export function CardTitle({ icon: Icon, children }: { icon: ComponentType<AntdIconProps>; children: ReactNode }) {
    return (
        <Flex align="center" gap={8}>
            <Icon aria-hidden="true" className="text-accent" />
            {children}
        </Flex>
    )
}
