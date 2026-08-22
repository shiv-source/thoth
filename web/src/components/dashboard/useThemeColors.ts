import { useMemo } from 'react'
import { theme } from 'antd'

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

// Chart colors come from the antd theme tokens (the single source of truth
// in theme.ts), so charts track the theme like every other component. The
// series hues are the categorical palette in index.css; the light theme
// never changes, so they are read once from :root.
export function useThemeColors(): ChartColors {
    const { token } = theme.useToken()
    return useMemo(() => {
        const css = getComputedStyle(document.documentElement)
        const read = (name: string, fallback: string) => css.getPropertyValue(name).trim() || fallback
        return {
            accent: token.colorPrimary,
            accentHover: token.colorPrimaryHover,
            subtle: token.colorTextSecondary,
            ink: token.colorText,
            surface: token.colorBgContainer,
            series: [
                read('--thoth-series-1', '#1677ff'),
                read('--thoth-series-2', '#0958d9'),
                read('--thoth-series-3', '#91caff'),
                read('--thoth-series-4', '#ffc53d')
            ]
        }
    }, [token])
}
