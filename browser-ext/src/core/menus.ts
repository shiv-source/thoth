import type { Draft } from './types'
import type { BrowserAPI } from './webext'

export const MENU_BOOKMARK = 'thoth-bookmark'
export const MENU_SELECTION = 'thoth-selection'
export const MENU_READLATER = 'thoth-readlater'
export const MENU_SUMMARIZE = 'thoth-summarize'

export interface MenuPageInfo {
    url: string
    title?: string
}

// registerMenus installs the right-click capture menus. The menu id drives the
// draft kind, so every menu shares one handler.
export async function registerMenus(api: BrowserAPI): Promise<void> {
    await api.contextMenus.removeAll()
    api.contextMenus.create({ id: MENU_BOOKMARK, title: 'Bookmark page to Thoth', contexts: ['page', 'link'] })
    api.contextMenus.create({ id: MENU_READLATER, title: 'Add to Thoth read later', contexts: ['page', 'link'] })
    api.contextMenus.create({ id: MENU_SELECTION, title: 'Save selection to Thoth', contexts: ['selection'] })
    api.contextMenus.create({ id: MENU_SUMMARIZE, title: 'Ask Thoth to summarize this page', contexts: ['page'] })
}

// draftForMenu maps a context-menu click to the pending capture draft. A
// selection click produces a selection draft (the quote); everything else is
// metadata-only — full-page text is captured only on the explicit summarize
// action or the popup's "include full page text" toggle (issue #176 #7).
export function draftForMenu(menuId: string, page: MenuPageInfo, selectionText?: string): Draft | null {
    switch (menuId) {
        case MENU_BOOKMARK:
            return { kind: 'bookmark', url: page.url, title: page.title || page.url }
        case MENU_READLATER:
            return { kind: 'readlater', url: page.url, title: page.title || page.url }
        case MENU_SELECTION:
            return { kind: 'selection', url: page.url, title: page.title || page.url, text: selectionText ?? '' }
        case MENU_SUMMARIZE:
            return { kind: 'summarize', url: page.url, title: page.title || page.url, text: '' }
        default:
            return null
    }
}
