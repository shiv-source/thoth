import type { ThothClient } from './api'
import type { BrowserAPI } from './webext'

// refreshBadge sets the toolbar badge to the unfiled inbox capture count — the
// glanceable "inbox has captures" signal (#4). Any server failure clears the
// badge silently; the popup's status line owns the error messaging.
export async function refreshBadge(api: BrowserAPI, client: ThothClient): Promise<void> {
    try {
        const count = await client.inboxCount()
        api.action.setBadgeBackgroundColor({ color: '#3b82f6' })
        api.action.setBadgeText({ text: count > 0 ? String(count) : '' })
        api.action.setTitle({
            title: count > 0 ? `Thoth — ${count} unfiled capture${count === 1 ? '' : 's'}` : 'Thoth Capture',
        })
    } catch {
        api.action.setBadgeText({ text: '' })
    }
}
