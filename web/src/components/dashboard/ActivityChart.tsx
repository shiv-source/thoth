import { useMemo } from 'react'
import { Bar } from 'react-chartjs-2'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import '../../utils/chart'
import { useThemeColors } from './useThemeColors'

// ActivityChart is a single-series mini bar chart (Chart.js via
// react-chartjs-2): notes created per day for the last N days (oldest
// first). One emerald hue, thin rounded bars, a built-in tooltip, a hidden
// value axis — the canvas carries the series description for screen readers.
export function ActivityChart({ counts }: { counts: number[] }) {
    // One row per bar, with the day labels computed from the current date —
    // the counts stay anchored to "today" without the mock going stale.
    const bars = useMemo(() => {
        const weekday = new Intl.DateTimeFormat('en-US', { weekday: 'short' })
        const date = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' })
        return counts.map((count, i) => {
            const d = new Date()
            d.setDate(d.getDate() - (counts.length - 1 - i))
            return { count, weekday: weekday.format(d), date: date.format(d) }
        })
    }, [counts])

    const colors = useThemeColors()
    const data: ChartData<'bar'> = {
        labels: bars.map((b) => b.weekday),
        datasets: [
            {
                data: bars.map((b) => b.count),
                backgroundColor: colors.accent,
                hoverBackgroundColor: colors.accentHover,
                borderRadius: 4,
                borderSkipped: false,
                maxBarThickness: 24
            }
        ]
    }
    const options: ChartOptions<'bar'> = {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        scales: {
            x: {
                grid: { display: false },
                border: { display: false },
                ticks: { color: colors.subtle, font: { size: 10 } }
            },
            y: { display: false, beginAtZero: true }
        },
        plugins: {
            legend: { display: false },
            tooltip: {
                displayColors: false,
                callbacks: {
                    title: () => '',
                    label: (item: TooltipItem<'bar'>) => {
                        const bar = bars[item.dataIndex]
                        if (!bar) return ''
                        return `${bar.date} · ${bar.count} note${bar.count === 1 ? '' : 's'}`
                    }
                }
            }
        }
    }

    return (
        <div className="h-24">
            <Bar
                data={data}
                options={options}
                role="img"
                aria-label={`Notes created per day for the last ${bars.length} days: ${bars[0]?.date} to ${
                    bars[bars.length - 1]?.date
                }`}
            />
        </div>
    )
}
