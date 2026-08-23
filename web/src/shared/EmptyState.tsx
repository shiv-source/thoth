import type { ReactNode } from 'react'

// EmptyState is the branded empty-state placeholder: a soft icon circle
// with a title, optional description, and optional action. It replaces
// antd's stock gray Empty so empty surfaces carry the product's tone.
export function EmptyState({
    icon,
    title,
    description,
    action,
    className = ''
}: {
    icon?: ReactNode
    title: string
    description?: string
    action?: ReactNode
    className?: string
}) {
    return (
        <div className={`flex flex-col items-center justify-center px-6 py-10 text-center ${className}`}>
            {icon && (
                <span className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-raised text-subtle">
                    {icon}
                </span>
            )}
            <p className="text-sm font-medium text-heading">{title}</p>
            {description && <p className="mt-1 max-w-sm text-xs leading-relaxed text-subtle">{description}</p>}
            {action && <div className="mt-4">{action}</div>}
        </div>
    )
}
