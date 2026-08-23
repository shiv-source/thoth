import { useMemo } from 'react'
import { Line } from 'react-chartjs-2'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import '../../utils/chart'
import { useThemeColors } from './useThemeColors'
import { chartDays } from './chartDays'
import { tooltipStyle, verticalGradient } from './chartTheme'

// ChatActivityChart is a single-series line chart: chat messages per day for
// the last N days (oldest first). One blue hue with a soft area fill,
// hover-only points, a light tooltip, a hidden value axis.
export function ChatActivityChart({ counts }: { counts: number[] }) {
    const days = useMemo(() => chartDays(counts.length), [counts.length])

    const colors = useThemeColors()
    const data: ChartData<'line'> = {
        labels: days.map((d) => d.weekday),
        datasets: [
            {
                data: counts,
                borderColor: colors.accent,
                backgroundColor: (context) =>
                    verticalGradient(context, 'rgba(22, 119, 255, 0.16)', 'rgba(22, 119, 255, 0)'),
                fill: true,
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
                ...tooltipStyle,
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
        <div className="h-28">
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
