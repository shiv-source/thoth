// EmptyState is the shared centered placeholder: an icon (emoji or node), a
// title, and an optional hint. className tweaks the layout for compact
// inline contexts (e.g. the notification panel).
export function EmptyState({
    icon,
    title,
    hint,
    className = ''
}: {
    icon: React.ReactNode
    title: string
    hint?: string
    className?: string
}) {
    return (
        <div className={`flex flex-1 flex-col items-center justify-center gap-3 text-subtle ${className}`}>
            <span className="text-3xl" aria-hidden="true">
                {icon}
            </span>
            <p className="text-sm">{title}</p>
            {hint && <p className="max-w-sm text-center text-xs">{hint}</p>}
        </div>
    )
}
