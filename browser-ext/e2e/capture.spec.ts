import { test, expect, chromium, type BrowserContext, type Page } from '@playwright/test'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const dirname = path.dirname(fileURLToPath(import.meta.url))
const EXT_PATH = path.resolve(dirname, '../dist/chrome')
const MOCK = 'http://127.0.0.1:8337'

let context: BrowserContext
let extensionId: string

test.beforeAll(async () => {
    context = await chromium.launchPersistentContext('', {
        channel: 'chromium',
        headless: true,
        args: [`--disable-extensions-except=${EXT_PATH}`, `--load-extension=${EXT_PATH}`],
    })
    // The MV3 service worker may spawn during launch; check the collection
    // before subscribing to the event so a fast spawn is never missed.
    const existing = context.serviceWorkers()
    const serviceWorker = existing[0] ?? (await context.waitForEvent('serviceworker', { timeout: 15000 }))
    extensionId = new URL(serviceWorker.url()).host
})

test.afterAll(async () => {
    await context.close()
})

// openPopup navigates to the extension's popup page directly (MV3 popups are
// pages; the toolbar click path is equivalent).
async function openPopup(): Promise<Page> {
    const page = await context.newPage()
    await page.goto(`chrome-extension://${extensionId}/popup.html`)
    return page
}

// connectToMock drives the popup's Server URL input to the mock server. Doing
// it through the UI exercises the connection-config feature end to end.
async function connectToMock(page: Page): Promise<void> {
    const input = page.getByLabel('Server URL')
    await input.fill(MOCK)
    await input.press('Enter')
    await expect(page.getByText(new RegExp(`Connected to ${MOCK.replace(/\./g, '\\.')}`))).toBeVisible()
}

async function captures(): Promise<unknown[]> {
    const res = await fetch(`${MOCK}/__captures`)
    const body = (await res.json()) as { captures: unknown[] }
    return body.captures
}

async function resetCaptures(): Promise<void> {
    await fetch(`${MOCK}/__captures`, { method: 'DELETE' })
}

async function markSaved(url: string): Promise<void> {
    await fetch(`${MOCK}/__saved`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
    })
}

test('connects to a server and captures a quick note into inbox/', async () => {
    await resetCaptures()
    const page = await openPopup()
    await connectToMock(page)

    await page.getByLabel('Text').fill('Quick thought')
    await page.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByText('Saved to inbox/quick-thought.md')).toBeVisible()
    const link = page.getByRole('link', { name: 'Open in Thoth' })
    await expect(link).toHaveAttribute('href', `${MOCK}/notes/inbox/quick-thought.md`)

    expect(await captures()).toMatchObject([{ kind: 'note', text: 'Quick thought', folder: 'inbox', tags: [] }])
    await page.close()
})

test('captures a bookmark line and opens it in the wiki', async () => {
    await resetCaptures()
    const page = await openPopup()
    await connectToMock(page)

    await page.getByText('Bookmark').click()
    await page.getByLabel('Title').fill('Go Channels')
    await page.getByLabel('URL', { exact: true }).fill('https://go.dev/blog/channels')
    await page.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByText('Saved to links/bookmarks.md')).toBeVisible()
    expect(await captures()).toMatchObject([
        { kind: 'bookmark', url: 'https://go.dev/blog/channels', title: 'Go Channels', category: 'unfiled' },
    ])
    await page.close()
})

test('blocks a duplicate bookmark with the "already saved → open it" state', async () => {
    await resetCaptures()
    await markSaved('https://go.dev/blog/channels')
    const page = await openPopup()
    await connectToMock(page)

    await page.getByText('Bookmark').click()
    await page.getByLabel('URL', { exact: true }).fill('https://go.dev/blog/channels')
    await page.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByText(/Already saved/)).toBeVisible()
    await expect(page.getByRole('link', { name: 'Open it' })).toHaveAttribute(
        'href',
        `${MOCK}/notes/links/bookmarks.md`
    )
    // Nothing was written: the dedup check short-circuited the POST.
    expect(await captures()).toEqual([])
    await page.close()
})

test('adds a link to the read-later queue', async () => {
    await resetCaptures()
    const page = await openPopup()
    await connectToMock(page)

    await page.getByText('Read later').click()
    await page.getByLabel('Title').fill('A long read')
    await page.getByLabel('URL', { exact: true }).fill('https://example.com/long-read')
    await page.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByText('Saved to links/read-later.md')).toBeVisible()
    expect(await captures()).toMatchObject([{ kind: 'readlater', url: 'https://example.com/long-read', title: 'A long read' }])
    await page.close()
})

test('validates input before hitting the server', async () => {
    await resetCaptures()
    const page = await openPopup()
    await connectToMock(page)

    // A note with no text is rejected client-side; the server is never called.
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByText('Text is required')).toBeVisible()
    expect(await captures()).toEqual([])
    await page.close()
})
