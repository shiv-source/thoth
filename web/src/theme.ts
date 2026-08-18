import type { ThemeConfig } from 'antd'

// Single source of truth for the enterprise SaaS theme (light-only).
// antd emits CSS variables from these tokens in cssVar mode (scoped under
// the ConfigProvider's css-var class — NOT on :root); the Tailwind tokens
// in index.css bridge to those variables with `@theme inline`, so each
// utility resolves the var where it is used, inside the scope.
export const antdTheme: ThemeConfig = {
    cssVar: {},
    // One antd version in the app — un-hashed classes keep the bundle
    // smaller and the DOM predictable.
    hashed: false,
    token: {
        colorPrimary: '#1677ff',
        borderRadius: 6,
        fontFamily: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif"
    },
    components: {
        Layout: {
            siderBg: '#ffffff',
            headerBg: '#ffffff',
            bodyBg: '#f5f5f5'
        },
        Menu: {
            itemBorderRadius: 6,
            iconSize: 16
        },
        // Settings rail — v6 Tabs tokens style the vertical items; the
        // active pill (background + weight) is the scoped CSS rule in
        // index.css, since tokens cover colors but not per-state fills.
        Tabs: {
            itemColor: 'rgba(0, 0, 0, 0.45)', // text-subtle
            itemHoverColor: 'rgba(0, 0, 0, 0.88)', // text-ink
            itemActiveColor: '#1677ff', // colorPrimary
            itemSelectedColor: '#1677ff', // colorPrimary
            inkBarColor: 'transparent', // the pill replaces the line indicator
            verticalItemMargin: '4px 0',
            verticalItemPadding: '12px 16px', // ≈44px touch target
            titleFontSize: 14
        },
        Button: {
            onlyIconSize: 16
        }
    }
}
