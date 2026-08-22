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
        Button: {
            onlyIconSize: 16
        }
    }
}
