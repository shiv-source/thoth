// chartDays — the date labels shared by the activity charts: one entry per
// day, oldest first, anchored to the current date (so the bars/lines stay
// "today" without the mock going stale).
export interface ChartDay {
    weekday: string
    date: string
}

export function chartDays(count: number): ChartDay[] {
    const weekday = new Intl.DateTimeFormat('en-US', { weekday: 'short' })
    const date = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' })
    return Array.from({ length: count }, (_, i) => {
        const d = new Date()
        d.setDate(d.getDate() - (count - 1 - i))
        return { weekday: weekday.format(d), date: date.format(d) }
    })
}
