import { ThothClient } from './api'
import { refreshBadge } from './badge'
import type { StorageLike } from './config'
import { resolveBaseUrl } from './config'
import { MENU_BOOKMARK, draftForMenu, registerMenus } from './menus'
import { capturePageText } from './page'
import { saveDraft } from './storage'
import type { Draft } from './types'
import type { BrowserAPI } from './webext'

export interface BackgroundDeps {
    api: BrowserAPI
    storage: StorageLike
}

// refresh re-probes the server and paints the badge. Called on install,
// startup, on a 5-minute alarm, and after every capture.
async function refresh(deps: BackgroundDeps): Promise<void> {
    const baseUrl = await resolveBaseUrl(deps.storage)
    if (!baseUrl) return
    await refreshBadge(deps.api, new ThothClient(baseUrl))
}

async function openPopup(api: BrowserAPI): Promise<boolean> {
    if (!api.action.openPopup) return false
    try {
        await api.action.openPopup()
        return true
    } catch {
        return false
    }
}

// draftFromMenuClick builds the pending capture draft for a context-menu
// click. The summarize menu grabs the page text up front (the page may change
// before the user reaches the popup); everything else stays metadata-only.
async function draftFromMenuClick(
    api: BrowserAPI,
    info: { menuItemId: string | number; pageUrl?: string; linkUrl?: string; selectionText?: string },
    tab?: { id?: number; title?: string },
): Promise<Draft | null> {
    const page = { url: info.linkUrl ?? info.pageUrl ?? '', title: tab?.title ?? '' }
    let draft = draftForMenu(String(info.menuItemId), page, info.selectionText)
    if (draft?.kind === 'summarize' && tab?.id != null) {
        draft = { ...draft, text: await capturePageText(api, tab.id) }
    }
    return draft
}

// initBackground wires the extension's background worker: context menus, the
// capture command, the badge, and the draft store that feeds the popup. The
// shared logic takes the API and storage as dependencies so tests drive it
// with fakes; the thin per-browser entries call it with the real globals.
export function initBackground(deps: BackgroundDeps): void {
    const { api } = deps

    api.runtime.onInstalled.addListener(() => {
        void registerMenus(api)
        void refresh(deps)
        api.alarms?.create('thoth-badge', { periodInMinutes: 5 })
    })
    api.runtime.onStartup?.addListener(() => {
        void refresh(deps)
    })
    api.alarms?.onAlarm.addListener((alarm) => {
        if (alarm.name === 'thoth-badge') void refresh(deps)
    })

    // A context-menu click stores a draft and opens the popup prefilled for
    // review (issue #176 #3). Browsers without action.openPopup fall back to a
    // badge hint that the user should click the toolbar icon.
    api.contextMenus.onClicked.addListener((info, tab) => {
        void handleMenuClick(deps, info, tab)
    })

    // Ctrl/Cmd+Shift+K captures the current page into a bookmark draft.
    api.commands?.onCommand.addListener((command) => {
        if (command === 'capture-page') void captureActiveTab(deps)
    })
}

async function handleMenuClick(
    deps: BackgroundDeps,
    info: { menuItemId: string | number; pageUrl?: string; linkUrl?: string; selectionText?: string },
    tab?: { id?: number; title?: string },
): Promise<void> {
    const draft = await draftFromMenuClick(deps.api, info, tab)
    if (!draft) return
    await saveDraft(deps.storage, draft)
    const opened = await openPopup(deps.api)
    if (!opened) {
        deps.api.action.setBadgeText({ text: '★' })
        deps.api.action.setTitle({ title: 'Thoth — click the toolbar icon to review your capture' })
    }
}

async function captureActiveTab(deps: BackgroundDeps): Promise<void> {
    const { api } = deps
    const [tab] = await api.tabs.query({ active: true, currentWindow: true })
    if (!tab?.url) return
    const draft = draftForMenu(MENU_BOOKMARK, { url: tab.url, title: tab.title ?? '' })
    if (!draft) return
    await saveDraft(deps.storage, draft)
    await openPopup(api)
}
