// IconButton is the subtle hover button used across the chrome (header
// actions, panel controls, dismiss buttons). Callers set aria-label and may
// override size via className.
export function IconButton({
    label,
    onClick,
    className = '',
    children
}: {
    label: string
    onClick: () => void
    className?: string
    children: React.ReactNode
}) {
    return (
        <button
            type="button"
            onClick={onClick}
            aria-label={label}
            className={`rounded-lg p-1.5 text-subtle transition hover:bg-raised hover:text-ink ${className}`}
        >
            {children}
        </button>
    )
}
