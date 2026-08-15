import { useEffect, useState } from 'react'

export interface ChartColors {
    accent: string
    accentHover: string
    subtle: string
    ink: string
    surface: string
    // Categorical series hues, in the validated order (blue, orange,
    // emerald, yellow) — see the series CSS variables in index.css.
    series: string[]
}

// Reads the theme's chart colors from the CSS variables so charts match
// the app palette.
export function chartColors(): ChartColors {
    const css = getComputedStyle(document.documentElement)
    const read = (name: string, fallback: string) => css.getPropertyValue(name).trim() || fallback
    return {
        accent: read('--thoth-accent', '#059669'),
        accentHover: read('--thoth-accent-hover', '#047857'),
        subtle: read('--thoth-subtle', '#64748b'),
        ink: read('--thoth-ink', '#0f172a'),
        surface: read('--thoth-surface', '#ffffff'),
        series: [
            read('--thoth-series-1', '#2a78d6'),
            read('--thoth-series-2', '#eb6834'),
            read('--thoth-series-3', '#059669'),
            read('--thoth-series-4', '#eda100')
        ]
    }
}

// The current chart colors, re-rendering the component when the OS theme
// flips so the Chart.js wrappers rebuild their data/options.
export function useThemeColors(): ChartColors {
    const [colors, setColors] = useState(chartColors)
    useEffect(() => {
        const media = window.matchMedia('(prefers-color-scheme: dark)')
        const onChange = () => setColors(chartColors())
        media.addEventListener('change', onChange)
        return () => media.removeEventListener('change', onChange)
    }, [])
    return colors
}
