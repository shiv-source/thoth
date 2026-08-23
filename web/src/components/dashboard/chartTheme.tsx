import type { ScriptableContext } from 'chart.js'

// verticalGradient builds a top-to-bottom gradient for scriptable fill
// colors (bars + line areas). chartArea is null before the chart lays out,
// so the solid bottom hue is the first-paint fallback.
export function verticalGradient<T extends 'bar' | 'line'>(
    context: ScriptableContext<T>,
    top: string,
    bottom: string
): string | CanvasGradient {
    const { chart } = context
    const { ctx, chartArea } = chart
    if (!chartArea) return bottom
    const g = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
    g.addColorStop(0, top)
    g.addColorStop(1, bottom)
    return g
}

// tooltipStyle is the shared light tooltip chrome: a white card with a hairline
// border and soft shadow that matches the surface language — no dark default.
export const tooltipStyle = {
    backgroundColor: '#ffffff',
    titleColor: '#0f172a',
    bodyColor: '#475569',
    borderColor: '#eef1f6',
    borderWidth: 1,
    boxShadow: '0 6px 16px rgba(15, 23, 42, 0.08)',
    padding: 10,
    cornerRadius: 8,
    boxPadding: 4,
    displayColors: false
} as const
