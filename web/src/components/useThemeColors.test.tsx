import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useThemeColors } from './useThemeColors'

function Probe() {
    const colors = useThemeColors()
    // A pipe-joined probe keeps the assertions string-typed — no JSON
    // round-trip, no any.
    const sample = `${colors.accent}|${colors.accentHover}|${colors.subtle}|${colors.surface}|${colors.series[3]}`
    return <span data-testid="colors">{sample}</span>
}

describe('useThemeColors', () => {
    it('derives chart colors from the antd theme tokens', () => {
        render(<Probe />)
        const parts = screen.getByTestId('colors').textContent.split('|')
        expect(parts[0]).toBe('#1677ff')
        expect(parts[1]).toBe('#4096ff')
        expect(parts[2]).toBe('rgba(0,0,0,0.65)')
        expect(parts[3]).toBe('#ffffff')
        // Series hues fall back to the categorical palette when the CSS
        // variables are absent (jsdom loads no stylesheet).
        expect(parts[4]).toBe('#ffc53d')
    })
})
