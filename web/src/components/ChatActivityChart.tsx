import { useMemo } from 'react'
import { Line } from 'react-chartjs-2'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import './chartSetup'
import { useThemeColors } from './useThemeColors'

// ChatActivityChart is a single-series line chart: chat messages per day for
// the last N days (oldest first). One emerald hue, a thin line with
// hover-only points, a hidden value axis.
export function ChatActivityChart({ counts }: { counts: number[] }) {
    const days = useMemo(() => {
        const weekday = new Intl.DateTimeFormat('en-US', { weekday: 'short' })
        const date = new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' })
        return counts.map((_, i) => {
            const d = new Date()
            d.setDate(d.getDate() - (counts.length - 1 - i))
            return { weekday: weekday.format(d), date: date.format(d) }
        })
    }, [counts])

    const colors = useThemeColors()
    const data: ChartData<'line'> = {
        labels: days.map((d) => d.weekday),
        datasets: [
            {
                data: counts,
                borderColor: colors.accent,
                backgroundColor: colors.accent,
                borderWidth: 2,
                pointRadius: 0,
                pointHoverRadius: 4,
                pointHoverBackgroundColor: colors.accent,
                pointHoverBorderColor: colors.surface,
                tension: 0.3
            }
        ]
    }
    const options: ChartOptions<'line'> = {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        scales: {
            x: {
                grid: { display: false },
                border: { display: false },
                ticks: { color: colors.subtle, font: { size: 10 }, maxTicksLimit: 7 }
            },
            y: { display: false, beginAtZero: true }
        },
        plugins: {
            legend: { display: false },
            tooltip: {
                displayColors: false,
                callbacks: {
                    title: () => '',
                    label: (item: TooltipItem<'line'>) => {
                        const day = days[item.dataIndex]
                        if (!day) return ''
                        const n = item.parsed.y
                        return `${day.date} · ${n} message${n === 1 ? '' : 's'}`
                    }
                }
            }
        }
    }

    return (
        <div className="h-24">
            <Line
                data={data}
                options={options}
                role="img"
                aria-label={`Chat messages per day for the last ${days.length} days: ${days[0]?.date} to ${
                    days[days.length - 1]?.date
                }`}
            />
        </div>
    )
}
