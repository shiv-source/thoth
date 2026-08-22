import { Bar } from 'react-chartjs-2'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import '../../utils/chart'
import { useThemeColors } from './useThemeColors'

// NotesByFolderChart compares note counts per top-level wiki folder
// (magnitude). Single emerald series, thin horizontal bars, folder names on
// the category axis, a hidden value axis.
export function NotesByFolderChart({ rows }: { rows: { folder: string; count: number }[] }) {
    const colors = useThemeColors()
    const data: ChartData<'bar'> = {
        labels: rows.map((r) => r.folder),
        datasets: [
            {
                data: rows.map((r) => r.count),
                backgroundColor: colors.accent,
                hoverBackgroundColor: colors.accentHover,
                borderRadius: 4,
                borderSkipped: false,
                maxBarThickness: 12
            }
        ]
    }
    const options: ChartOptions<'bar'> = {
        indexAxis: 'y',
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        scales: {
            x: { display: false, beginAtZero: true },
            y: {
                grid: { display: false },
                border: { display: false },
                ticks: { color: colors.subtle, font: { size: 11 } }
            }
        },
        plugins: {
            legend: { display: false },
            tooltip: {
                displayColors: false,
                callbacks: {
                    title: () => '',
                    label: (item: TooltipItem<'bar'>) => {
                        const n = item.parsed.x
                        return `${item.label} · ${n} note${n === 1 ? '' : 's'}`
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
                aria-label={`Notes per wiki folder: ${rows.map((r) => `${r.folder} ${r.count}`).join(', ')}`}
            />
        </div>
    )
}
