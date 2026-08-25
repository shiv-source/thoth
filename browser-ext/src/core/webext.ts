import type { StorageLike } from './config'

export interface TabInfo {
    id?: number
    url?: string
    title?: string
}

export interface ContextMenuClickInfo {
    menuItemId: string | number
    pageUrl?: string
    linkUrl?: string
    selectionText?: string
}

export interface MenuItemSpec {
    id: string
    title: string
    contexts: Array<'page' | 'link' | 'selection'>
}

// BrowserAPI is the subset of the browser extension API the extension uses,
// shaped so both Chrome (chrome.*) and Firefox (browser.*) satisfy it. Shared
// logic takes it as a parameter so tests inject a fake instead of loading a
// real browser global.
export interface BrowserAPI {
    runtime: {
        onInstalled: { addListener(listener: () => void): void }
        onStartup?: { addListener(listener: () => void): void }
    }
    contextMenus: {
        removeAll(): Promise<void>
        create(spec: MenuItemSpec): void
        onClicked: { addListener(listener: (info: ContextMenuClickInfo, tab?: TabInfo) => void): void }
    }
    action: {
        setBadgeText(details: { text: string }): void
        setBadgeBackgroundColor(details: { color: string }): void
        setTitle(details: { title: string }): void
        openPopup?(): Promise<void>
    }
    tabs: {
        query(queryInfo: { active: boolean; currentWindow: boolean }): Promise<TabInfo[]>
        create(details: { url: string }): Promise<unknown>
    }
    scripting: {
        executeScript(details: { target: { tabId: number }; func: () => string }): Promise<Array<{ result?: string }>>
    }
    storage: {
        local: {
            get(key: string): Promise<Record<string, string>>
            set(items: Record<string, string>): Promise<void>
            remove(key: string): Promise<void>
        }
    }
    alarms?: {
        create(name: string, info: { periodInMinutes: number }): void
        onAlarm: { addListener(listener: (alarm: { name: string }) => void): void }
    }
    commands?: {
        onCommand: { addListener(listener: (command: string) => void): void }
    }
}

const globalScope = globalThis as unknown as { browser?: BrowserAPI; chrome?: BrowserAPI }

// ext is the current browser's extension API. Firefox exposes browser.*;
// Chrome exposes chrome.*. Importing this module throws outside a browser
// extension context — only the browser entry points (background + popup)
// import it; shared logic takes the API as a parameter instead.
export const ext: BrowserAPI = (() => {
    if (globalScope.browser) return globalScope.browser
    if (globalScope.chrome) return globalScope.chrome
    throw new Error('no browser extension API (browser.*/chrome.*) found')
})()

// webextStorage adapts ext.storage.local to the StorageLike seam the config
// and draft helpers consume.
export const webextStorage: StorageLike = {
    async get(key: string): Promise<string | undefined> {
        const record = await ext.storage.local.get(key)
        return record[key]
    },
    async set(key: string, value: string): Promise<void> {
        await ext.storage.local.set({ [key]: value })
    },
    async remove(key: string): Promise<void> {
        await ext.storage.local.remove(key)
    },
}
