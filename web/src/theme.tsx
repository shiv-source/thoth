import type { ThemeConfig } from 'antd'

// Single source of truth for the Thoth design system (light-only).
//
// Token strategy:
//   - Brand: #1677ff stays the anchor hue; every map token (hover / active /
//     bg / border / text) derives from it, so no component hardcodes a
//     variant.
//   - Neutrals: a cool gray ramp (slate) instead of antd's warm gray — layout
//     #f5f7fa, border #e3e7ee — so surfaces read as precise, not papery.
//   - Calibration: radius 8/12, control height 36, a soft 3px blue focus
//     ring, and layered shadows give the chrome a premium-but-restrained feel.
//   - Dark mode is not shipped, but every value lives here, so a dark
//     ThemeConfig is a pure token flip — no component needs to change.
//
// antd emits these as --ant-* CSS variables in cssVar mode, scoped under the
// ConfigProvider's wrapper class (NOT :root); the Tailwind tokens in
// index.css bridge to them with `@theme inline` so each utility resolves the
// var where it is used, inside the scope.
export const antdTheme: ThemeConfig = {
    cssVar: {},
    // One antd version in the app — un-hashed classes keep the bundle
    // smaller and the DOM predictable.
    hashed: false,

    token: {
        // Brand — antd blue, kept as the anchor hue.
        colorPrimary: '#1677ff',
        colorLink: '#1677ff',

        // Semantics (antd defaults, kept explicit).
        colorSuccess: '#52c41a',
        colorWarning: '#faad14',
        colorError: '#ff4d4d',
        colorInfo: '#1677ff',

        // Neutral ramp — cool gray for enterprise precision.
        colorBgLayout: '#f5f7fa',
        colorBgContainer: '#ffffff',
        colorBgElevated: '#ffffff',
        colorBorder: '#e3e7ee',
        colorBorderSecondary: '#eef1f6',
        colorSplit: 'rgba(15, 23, 42, 0.06)',
        colorFillTertiary: 'rgba(15, 23, 42, 0.04)',
        colorFillSecondary: 'rgba(15, 23, 42, 0.06)',
        colorText: 'rgba(15, 23, 42, 0.88)',
        colorTextSecondary: 'rgba(15, 23, 42, 0.56)',
        colorTextTertiary: 'rgba(15, 23, 42, 0.4)',
        colorTextQuaternary: 'rgba(15, 23, 42, 0.25)',
        colorTextHeading: '#0f172a',
        colorTextDescription: 'rgba(15, 23, 42, 0.56)',

        // Focus ring — soft blue, generous.
        controlOutline: 'rgba(22, 119, 255, 0.12)',
        controlOutlineWidth: 3,

        // Radius — calibrated, not bubbly.
        borderRadius: 8,
        borderRadiusLG: 12,
        borderRadiusSM: 6,
        borderRadiusXS: 4,

        // Type.
        fontFamily: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
        fontFamilyCode: "ui-monospace, 'SF Mono', SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace",
        fontSize: 14,
        fontSizeLG: 15,
        fontSizeHeading1: 30,
        fontSizeHeading2: 24,
        fontSizeHeading3: 20,
        fontSizeHeading4: 17,
        fontSizeHeading5: 15,

        // Controls — slightly taller than default for calmer hit targets.
        controlHeight: 36,
        controlHeightLG: 44,
        controlHeightSM: 28,

        // Elevation — layered, soft.
        boxShadow: '0 1px 2px rgba(15, 23, 42, 0.04), 0 2px 8px rgba(15, 23, 42, 0.04)',
        boxShadowSecondary: '0 6px 16px rgba(15, 23, 42, 0.08), 0 1px 3px rgba(15, 23, 42, 0.06)',
        boxShadowTertiary: '0 1px 2px rgba(15, 23, 42, 0.03), 0 4px 12px rgba(15, 23, 42, 0.04)',

        // Motion — one set of timings app-wide.
        motionDurationFast: '0.12s',
        motionDurationMid: '0.2s',
        motionDurationSlow: '0.3s'
    },

    components: {
        Layout: {
            bodyBg: '#f5f7fa',
            headerBg: '#ffffff',
            headerColor: '#0f172a',
            headerHeight: 56,
            headerPadding: '0 20px',
            siderBg: '#ffffff',
            lightSiderBg: '#ffffff'
        },
        Menu: {
            iconSize: 16,
            itemHeight: 40,
            itemBorderRadius: 8,
            itemMarginBlock: 2,
            itemColor: 'rgba(15, 23, 42, 0.6)',
            itemHoverBg: 'rgba(15, 23, 42, 0.04)',
            itemHoverColor: '#0f172a',
            itemSelectedBg: 'rgba(22, 119, 255, 0.1)',
            itemSelectedColor: '#1677ff',
            collapsedIconSize: 18,
            groupTitleColor: 'rgba(15, 23, 42, 0.4)'
        },
        Button: {
            fontWeight: 500,
            primaryShadow: '0 1px 2px rgba(22, 119, 255, 0.2)',
            onlyIconSize: 16,
            contentFontSize: 14,
            defaultBorderColor: '#dde2ea',
            defaultHoverBorderColor: '#c7cede',
            textHoverBg: 'rgba(15, 23, 42, 0.04)'
        },
        Card: {
            headerFontSize: 15,
            headerFontSizeSM: 14
        },
        Table: {
            headerBg: '#f7f8fb',
            headerColor: '#5a6474',
            headerSplitColor: 'transparent',
            rowHoverBg: 'rgba(22, 119, 255, 0.04)'
        },
        Statistic: {
            contentFontSize: 28,
            titleFontSize: 13
        },
        Tree: {
            titleHeight: 32,
            nodeHoverBg: 'rgba(15, 23, 42, 0.04)',
            nodeSelectedBg: 'rgba(22, 119, 255, 0.1)',
            nodeSelectedColor: '#1677ff'
        },
        Alert: {
            borderRadius: 10
        },
        Input: {
            hoverBorderColor: '#b8c1d1',
            activeBorderColor: '#1677ff',
            activeShadow: '0 0 0 3px rgba(22, 119, 255, 0.12)'
        }
    }
}
