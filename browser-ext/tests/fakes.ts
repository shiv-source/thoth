import type { StorageLike } from '../src/core/config'
import type { BrowserAPI, ContextMenuClickInfo, MenuItemSpec, TabInfo } from '../src/core/webext'

type Listener<Args extends unknown[]> = (...args: Args) => void | Promise<void>

// MemoryStorage is an in-memory StorageLike for config/draft tests.
export function memoryStorage(seed: Record<string, string> = {}): StorageLike & { data: Record<string, string> } {
    const data = { ...seed }
    return {
        data,
        async get(key) {
            return data[key]
        },
        async set(key, value) {
            data[key] = value
        },
        async remove(key) {
            delete data[key]
        },
    }
}

export interface FakeAPI {
    api: BrowserAPI
    installed: Listener<[]>[]
    started: Listener<[]>[]
    menuCreated: MenuItemSpec[]
    badge: Array<{ type: 'text' | 'color' | 'title'; value: string }>
    tabsCreated: string[]
    queried: boolean
    scripted: number[]
    openedPopup: boolean
    alarmNames: string[]
    permissionRequests: Array<{ origins: string[] }>
    fireInstalled(): Promise<void>
    fireStarted(): Promise<void>
    fireMenuClick(info: Partial<ContextMenuClickInfo>, tab?: Partial<TabInfo>): Promise<void>
    fireCommand(command: string): Promise<void>
    fireAlarm(name: string): Promise<void>
}

// fakeBrowserAPI is a scriptable BrowserAPI: listeners are captured so tests
// can fire install/startup/menu/command/alarm events, and every method call is
// recorded for assertions. storage.local is a stub — callers inject their own
// StorageLike into the code under test.
export function fakeBrowserAPI(): FakeAPI {
    const installed: Listener<[]>[] = []
    const started: Listener<[]>[] = []
    const menuHandlers: Array<(info: ContextMenuClickInfo, tab?: TabInfo) => void | Promise<void>> = []
    const commandHandlers: Array<(command: string) => void | Promise<void>> = []
    const alarmHandlers: Array<(alarm: { name: string }) => void> = []
    const menuCreated: MenuItemSpec[] = []
    const badge: Array<{ type: 'text' | 'color' | 'title'; value: string }> = []
    const tabsCreated: string[] = []
    const scripted: number[] = []
    const alarmNames: string[] = []
    const permissionRequests: Array<{ origins: string[] }> = []
    let queried = false
    let openedPopup = false

    const api: BrowserAPI = {
        runtime: {
            onInstalled: { addListener: (fn) => void installed.push(fn) },
            onStartup: { addListener: (fn) => void started.push(fn) },
        },
        contextMenus: {
            removeAll: async () => {},
            create: (spec) => void menuCreated.push(spec),
            onClicked: { addListener: (fn) => void menuHandlers.push(fn) },
        },
        action: {
            setBadgeText: (d) => void badge.push({ type: 'text', value: d.text }),
            setBadgeBackgroundColor: (d) => void badge.push({ type: 'color', value: d.color }),
            setTitle: (d) => void badge.push({ type: 'title', value: d.title }),
            openPopup: async () => {
                openedPopup = true
            },
        },
        tabs: {
            query: async () => {
                queried = true
                return [{ id: 7, url: 'https://example.com', title: 'Example' }]
            },
            create: async (d) => {
                tabsCreated.push(d.url)
                return {}
            },
        },
        scripting: {
            executeScript: async (d) => {
                scripted.push(d.target.tabId)
                return [{ result: 'page inner text' }]
            },
        },
        storage: {
            local: {
                get: async (key) => ({ [key]: '' }),
                set: async () => {},
                remove: async () => {},
            },
        },
        alarms: {
            create: (name) => void alarmNames.push(name),
            onAlarm: { addListener: (fn) => void alarmHandlers.push(fn) },
        },
        commands: {
            onCommand: { addListener: (fn) => void commandHandlers.push(fn) },
        },
        permissions: {
            // Requests are granted by default; tests override to simulate a
            // denial. Every request is recorded for assertions.
            request: async (perms): Promise<boolean> => {
                permissionRequests.push(perms)
                return true
            },
        },
    }

    return {
        api,
        installed,
        started,
        menuCreated,
        badge,
        tabsCreated,
        // Live getters: the setters mutate the closure booleans after the
        // object is returned, so a copied value would never update.
        get queried() {
            return queried
        },
        get openedPopup() {
            return openedPopup
        },
        scripted,
        alarmNames,
        permissionRequests,
        async fireInstalled() {
            for (const fn of installed) await Promise.resolve(fn())
        },
        async fireStarted() {
            for (const fn of started) await Promise.resolve(fn())
        },
        async fireMenuClick(info: Partial<ContextMenuClickInfo>, tab?: Partial<TabInfo>) {
            const full: ContextMenuClickInfo = { menuItemId: 'x', pageUrl: 'https://example.com', ...info }
            for (const fn of menuHandlers) await Promise.resolve(fn(full, tab))
        },
        async fireCommand(command: string) {
            for (const fn of commandHandlers) await Promise.resolve(fn(command))
        },
        async fireAlarm(name: string) {
            for (const fn of alarmHandlers) await Promise.resolve(fn({ name }))
        },
    }
}
