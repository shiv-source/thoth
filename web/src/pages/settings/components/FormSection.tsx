import type { ComponentType, ReactNode } from 'react'
import type { AntdIconProps } from '@ant-design/icons/lib/components/AntdIcon'
import { SectionHeading } from './SectionHeading'

// FormSection is the enterprise grouping block for a settings card: an
// icon'd heading with an optional one-line description, then the fields.
// Every settings page uses the same block, so the four tabs share one
// vertical rhythm and read as a single system rather than four ad-hoc
// pages.
export function FormSection({
    icon,
    title,
    description,
    children
}: {
    icon: ComponentType<AntdIconProps>
    title: string
    description?: string
    children: ReactNode
}) {
    return (
        <section>
            <SectionHeading icon={icon}>{title}</SectionHeading>
            {description && <p className="mb-4 mt-1 text-sm leading-relaxed text-subtle">{description}</p>}
            <div className="space-y-5">{children}</div>
        </section>
    )
}
