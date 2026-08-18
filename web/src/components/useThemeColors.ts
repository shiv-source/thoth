import { useMemo } from 'react'

export interface ChartColors {
    accent: string
    accentHover: string
    subtle: string
    ink: string
    surface: string
    // Categorical series hues, in the validated order — see the series
    // CSS variables in index.css.
    series: string[]
}

// Reads the theme's chart colors from the CSS variables so charts match
// the app palette. Light theme only, so values are read once.
export function chartColors(): ChartColors {
    const css = getComputedStyle(document.documentElement)
    const read = (name: string, fallback: string) => css.getPropertyValue(name).trim() || fallback
    return {
        accent: read('--thoth-accent', '#1677ff'),
        accentHover: read('--thoth-accent-hover', '#4096ff'),
        subtle: read('--thoth-subtle', 'rgba(0, 0, 0, 0.45)'),
        ink: read('--thoth-ink', 'rgba(0, 0, 0, 0.88)'),
        surface: read('--thoth-surface', '#ffffff'),
        series: [
            read('--thoth-series-1', '#1677ff'),
            read('--thoth-series-2', '#0958d9'),
            read('--thoth-series-3', '#91caff'),
            read('--thoth-series-4', '#ffc53d')
        ]
    }
}

// The chart colors for the current theme. With no dark mode the palette
// never changes, so this is a stable value for the Chart.js wrappers.
export function useThemeColors(): ChartColors {
    return useMemo(chartColors, [])
}
