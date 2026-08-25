import type { BrowserAPI } from './webext'

// capturePageText grabs a tab's visible text via the scripting API. It works
// because the user's invocation (context-menu click, action, command) granted
// activeTab access to that tab. Returns '' when the page is not scriptable
// (chrome:// pages, PDFs, …).
export async function capturePageText(api: BrowserAPI, tabId: number): Promise<string> {
    try {
        const results = await api.scripting.executeScript({
            target: { tabId },
            func: () => document.body?.innerText ?? '',
        })
        return results[0]?.result ?? ''
    } catch {
        return ''
    }
}
