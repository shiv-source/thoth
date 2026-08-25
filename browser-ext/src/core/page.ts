import { MAX_CAPTURE_TEXT } from './format'
import type { BrowserAPI } from './webext'

// capturePageText grabs a tab's visible text via the scripting API. It works
// because the user's invocation (context-menu click, action, command) granted
// activeTab access to that tab. Returns '' when the page is not scriptable
// (chrome:// pages, PDFs, …). The result is truncated to MAX_CAPTURE_TEXT so a
// huge page never trips the server's body guard.
export async function capturePageText(api: BrowserAPI, tabId: number): Promise<string> {
    try {
        const results = await api.scripting.executeScript({
            target: { tabId },
            func: () => document.body?.innerText ?? '',
        })
        const text = results[0]?.result ?? ''
        return text.length > MAX_CAPTURE_TEXT ? text.slice(0, MAX_CAPTURE_TEXT) : text
    } catch {
        return ''
    }
}
