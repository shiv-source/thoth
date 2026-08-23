import { Doughnut } from 'react-chartjs-2'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import '../../utils/chart'
import { useThemeColors } from './useThemeColors'
import { tooltipStyle } from './chartTheme'

// NotesByKindChart is a doughnut of note counts by kind (part-to-whole).
// The categorical hues come from the validated series palette; a 2px
// surface gap separates the segments (the CVD secondary encoding), and the
// legend list below carries the identity so color is never the only
// encoding.
export function NotesByKindChart({ slices }: { slices: { kind: string; count: number }[] }) {
    const colors = useThemeColors()
    const data: ChartData<'doughnut'> = {
        labels: slices.map((s) => s.kind),
        datasets: [
            {
                data: slices.map((s) => s.count),
                backgroundColor: colors.series.slice(0, slices.length),
                borderColor: colors.surface,
                borderWidth: 2,
                hoverOffset: 4
            }
        ]
    }
    const options: ChartOptions<'doughnut'> = {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        cutout: '68%',
        plugins: {
            legend: { display: false },
            tooltip: {
                ...tooltipStyle,
                callbacks: {
                    label: (item: TooltipItem<'doughnut'>) => `${item.label} · ${item.parsed} notes`
                }
            }
        }
    }

    return (
        <div>
            <div className="h-32">
                <Doughnut
                    data={data}
                    options={options}
                    role="img"
                    aria-label={`Notes by kind: ${slices.map((s) => `${s.kind} ${s.count}`).join(', ')}`}
                />
            </div>
            <ul className="mt-3 space-y-1.5">
                {slices.map((s, i) => (
                    <li key={s.kind} className="flex items-center gap-2 text-sm text-ink">
                        <span
                            aria-hidden="true"
                            className="h-2.5 w-2.5 shrink-0 rounded-full"
                            style={{ backgroundColor: `var(--thoth-series-${i + 1})` }}
                        />
                        {s.kind}
                        <span className="ml-auto font-mono text-xs text-subtle">{s.count}</span>
                    </li>
                ))}
            </ul>
        </div>
    )
}
