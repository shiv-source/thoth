import '@testing-library/jest-dom/vitest'

// jsdom has no ResizeObserver; antd's popover/tooltip machinery observes
// resize, so a stub keeps the tree from throwing under test. No-op under the
// node-environment suites.
class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
}
globalThis.ResizeObserver ??= ResizeObserverStub as unknown as typeof ResizeObserver

// jsdom has no matchMedia; antd's responsive hooks (Grid, Segmented, …) read
// it. The stub never matches, keeping theme/media queries inert under test.
// This file also loads for node-environment suites, where window does not
// exist.
if (typeof window !== 'undefined') {
    Object.defineProperty(window, 'matchMedia', {
        configurable: true,
        value: (query: string) => ({
            matches: false,
            media: query,
            onchange: null,
            addEventListener: () => {},
            removeEventListener: () => {},
            addListener: () => {},
            removeListener: () => {},
            dispatchEvent: () => false
        })
    })
}
