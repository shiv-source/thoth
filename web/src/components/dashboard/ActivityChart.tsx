import { useMemo } from 'react'
import { Bar } from 'react-chartjs-2'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import '../../utils/chart'
import { useThemeColors } from './useThemeColors'
import { chartDays } from './chartDays'
import { tooltipStyle, verticalGradient } from './chartTheme'

// ActivityChart is a single-series mini bar chart (Chart.js via
// react-chartjs-2): notes created per day for the last N days (oldest
// first). One blue hue with a vertical gradient, thin rounded bars, a built-in
// light tooltip, a hidden value axis — the canvas carries the series
// description for screen readers.
export function ActivityChart({ counts }: { counts: number[] }) {
    const bars = useMemo(() => chartDays(counts.length), [counts.length])

    const colors = useThemeColors()
    const data: ChartData<'bar'> = {
        labels: bars.map((b) => b.weekday),
        datasets: [
            {
                data: counts,
                backgroundColor: (context) => verticalGradient(context, colors.accentHover, colors.accent),
                hoverBackgroundColor: colors.accent,
                borderRadius: 6,
                borderSkipped: false,
                maxBarThickness: 26
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
                ...tooltipStyle,
                callbacks: {
                    title: () => '',
                    label: (item: TooltipItem<'bar'>) => {
                        const bar = bars[item.dataIndex]
                        if (!bar) return ''
                        return `${bar.date} · ${item.parsed.y} note${item.parsed.y === 1 ? '' : 's'}`
                    }
                }
            }
        }
    }

    return (
        <div className="h-28">
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
