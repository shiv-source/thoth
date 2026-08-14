import '@testing-library/jest-dom/vitest'

// jsdom does not implement scrollIntoView; ChatPanel scrolls to the latest message.
window.HTMLElement.prototype.scrollIntoView = function () {}
