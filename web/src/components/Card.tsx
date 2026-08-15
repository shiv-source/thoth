import type { ReactNode } from 'react'

// Card is the app-wide section idiom: a surface panel with an uppercase
// kicker title. Views compose them on the bg-app background.
export function Card({ title, children }: { title: string; children: ReactNode }) {
    return (
        <section className="rounded-xl border border-line bg-surface p-4">
            <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-subtle">{title}</h2>
            {children}
        </section>
    )
}
