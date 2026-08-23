import type { ReactNode } from 'react'

// SectionHeader is the page-section kicker: a small accent tick and an
// uppercase micro-label, used to separate blocks on a page (dashboard
// Overview/Insights, settings sections).
export function SectionHeader({ children }: { children: ReactNode }) {
    return (
        <div className="flex items-center gap-2">
            <span aria-hidden="true" className="h-3.5 w-1 rounded-full bg-accent" />
            <h2 className="text-xs font-semibold uppercase tracking-wider text-faint">{children}</h2>
        </div>
    )
}
