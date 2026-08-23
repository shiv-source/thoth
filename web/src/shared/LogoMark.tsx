import { useId } from 'react'
import { theme } from 'antd'

// LogoMark is the Thoth owl mark: an accent tile with a white owl silhouette
// and an amber beak. Colors come from the antd tokens (hover→active blue) so
// the mark tracks the theme like every other component; the gradient id is
// unique per mount so multiple marks on one page never clash. Decorative by
// default — pair it with Logo's wordmark for the accessible name.
export function LogoMark({ size = 28 }: { size?: number }) {
    const { token } = theme.useToken()
    const id = useId()
    const gid = `thoth-grad-${id}`

    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 48 48"
            fill="none"
            aria-hidden="true"
            className="shrink-0 drop-shadow-sm"
        >
            <defs>
                <linearGradient id={gid} x1="0" y1="0" x2="48" y2="48" gradientUnits="userSpaceOnUse">
                    <stop stopColor={token.colorPrimaryHover} />
                    <stop offset="1" stopColor={token.colorPrimaryActive} />
                </linearGradient>
            </defs>
            <rect width="48" height="48" rx="13" fill={`url(#${gid})`} />
            <path
                d="M24 4 33 12 39.5 10.5 36 19c3 4 4 8.5 3.5 12.5C38.5 38 32.5 43 24 43S9.5 38 8.5 31.5C8 27.5 9 23 12 19L8.5 10.5 15 12Z"
                fill="#fff"
            />
            <circle cx="17.5" cy="25.5" r="5.6" fill={token.colorPrimaryActive} />
            <circle cx="30.5" cy="25.5" r="5.6" fill={token.colorPrimaryActive} />
            <circle cx="19" cy="23.6" r="1.8" fill="#fff" />
            <circle cx="32" cy="23.6" r="1.8" fill="#fff" />
            <path d="M24 31.5 20.6 36.5h6.8Z" fill="#ffc53d" />
        </svg>
    )
}
