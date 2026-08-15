import '@testing-library/jest-dom/vitest'

// jsdom does not implement scrollIntoView; ChatPanel scrolls to the latest message.
window.HTMLElement.prototype.scrollIntoView = function () {}

// Radix tooltips (Popper) observe resize; jsdom has no ResizeObserver, so
// opening a tooltip would throw and unmount the tree under test.
class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
}
globalThis.ResizeObserver ??= ResizeObserverStub as unknown as typeof ResizeObserver

// jsdom has no matchMedia; ActivityChart watches prefers-color-scheme so the
// chart flips with the theme. The stub never matches, which keeps theme
// checks inert under test.
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
