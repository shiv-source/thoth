import type { ReactNode } from 'react'
import { Card } from 'antd'

// ChartCard is the Insights widget shell: a titled card wrapping one chart
// plus its mock-data footnote, so the four Insights cards stay identical.
export function ChartCard({ title, note, children }: { title: string; note: string; children: ReactNode }) {
    return (
        <Card size="small" title={title}>
            {children}
            <p className="mt-3 text-xs text-subtle">{note}</p>
        </Card>
    )
}
