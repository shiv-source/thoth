import { LogoMark } from './LogoMark'

// Logo is the brand lockup: the owl mark plus the Fraunces wordmark. The
// mark is decorative; the wordmark carries the accessible name.
export function Logo({
    size = 28,
    wordmark = true,
    className = ''
}: {
    size?: number
    wordmark?: boolean
    className?: string
}) {
    return (
        <span className={`inline-flex items-center gap-2.5 ${className}`}>
            <LogoMark size={size} />
            {wordmark && <span className="font-display text-lg font-semibold tracking-tight text-heading">Thoth</span>}
        </span>
    )
}
