import type { ThemeConfig } from 'antd'

// The popup's antd theme — a compact mirror of the dashboard's design system
// (web/src/theme.tsx): the same blue anchor hue and cool-neutral ramp, so a
// capture form in the extension reads as part of the same product.
export const popupTheme: ThemeConfig = {
    cssVar: {},
    hashed: false,
    token: {
        colorPrimary: '#1677ff',
        colorLink: '#1677ff',
        colorBgLayout: '#f5f7fa',
        colorBgContainer: '#ffffff',
        colorTextBase: '#1f2430',
        colorTextSecondary: '#667085',
        colorBorder: '#e3e7ee',
        borderRadius: 8,
        controlHeight: 34,
        fontFamily:
            "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    },
}
