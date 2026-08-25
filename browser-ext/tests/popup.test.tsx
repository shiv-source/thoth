// @vitest-environment jsdom
import { ConfigProvider } from 'antd'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PopupApp } from '../src/popup/App'
import { popupTheme } from '../src/popup/theme'
import { BASE_URL_KEY } from '../src/core/config'
import { fakeBrowserAPI, memoryStorage } from './fakes'

// stubServer answers the routes the popup hits during a test. Routes not
// listed fall back to a non-ok health probe (so discovery moves on).
function stubServer(routes: Record<string, unknown>) {
    vi.stubGlobal(
        'fetch',
        vi.fn(async (url: string) => {
            const key = Object.keys(routes).find((k) => url.includes(k))
            if (!key) return { ok: false, status: 404, json: async () => ({}) }
            const body = routes[key]
            return { ok: true, status: 200, json: async () => body }
        })
    )
}

function renderPopup(storage = memoryStorage()) {
    const ext = fakeBrowserAPI()
    const view = render(
        <ConfigProvider theme={popupTheme}>
            <PopupApp storage={storage} ext={ext.api} />
        </ConfigProvider>
    )
    return { view, ext, storage }
}

afterEach(() => {
    vi.unstubAllGlobals()
})

const settingsBody = {
    wiki_path: '~/.thoth/wiki',
    wiki_folders: ['inbox', 'knowledge', 'links', 'projects'],
    model: '',
    providers: {},
    context_injection: false,
    conversation_retention_days: 7,
}

describe('PopupApp', () => {
    it('connects and shows the capture form when a server is found', async () => {
        stubServer({ '/api/v1/health': { status: 'ok' }, '/api/v1/settings': settingsBody })
        renderPopup()
        expect(await screen.findByText(/Connected to/)).toBeInTheDocument()
        expect(screen.getByLabelText('Capture kind')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /Save/ })).toBeInTheDocument()
        await waitFor(() => expect(screen.getByLabelText('Folder')).toBeInTheDocument())
    })

    it('shows the "start thoth" state when no server answers', async () => {
        stubServer({})
        renderPopup()
        expect(await screen.findByText(/Thoth is not running/)).toBeInTheDocument()
        expect(screen.queryByLabelText('Capture kind')).not.toBeInTheDocument()
    })

    it('prefills a pending bookmark draft for review', async () => {
        stubServer({ '/api/v1/health': { status: 'ok' }, '/api/v1/settings': settingsBody })
        const storage = memoryStorage({
            [BASE_URL_KEY]: 'http://127.0.0.1:8333',
            'thoth:draft': JSON.stringify({
                kind: 'bookmark',
                url: 'https://go.dev/blog/channels',
                title: 'Go Channels',
                category: 'reference',
            }),
        })
        renderPopup(storage)
        expect(await screen.findByLabelText('Title')).toHaveValue('Go Channels')
        expect(screen.getByLabelText('URL')).toHaveValue('https://go.dev/blog/channels')
        // Bookmark fields (category/reason) are visible; the note text field is not.
        expect(screen.getByLabelText('Category')).toBeInTheDocument()
        expect(screen.queryByLabelText('Text')).not.toBeInTheDocument()
    })

    it('prefills a selection draft as a quoted body with a read-only URL', async () => {
        stubServer({ '/api/v1/health': { status: 'ok' }, '/api/v1/settings': settingsBody })
        const storage = memoryStorage({
            'thoth:draft': JSON.stringify({
                kind: 'selection',
                url: 'https://example.com/article',
                title: 'Article',
                text: 'the quote',
            }),
        })
        renderPopup(storage)
        const urlInput = await screen.findByLabelText('URL')
        expect(urlInput).toHaveValue('https://example.com/article')
        expect(urlInput).toHaveAttribute('readonly')
        expect(screen.getByLabelText('Text')).toHaveValue('> the quote\n\n> — [Article](https://example.com/article)')
    })

    it('saves a quick note through the capture endpoint and offers Open in Thoth', async () => {
        stubServer({
            '/api/v1/health': { status: 'ok' },
            '/api/v1/settings': settingsBody,
            '/api/v1/capture': { path: 'inbox/quick-thought.md', title: 'Quick thought', type: 'inbox' },
        })
        const { storage } = renderPopup()
        await screen.findByLabelText('Text')
        await userEvent.type(screen.getByLabelText('Text'), 'Quick thought')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))

        expect(await screen.findByText('Saved to inbox/quick-thought.md')).toBeInTheDocument()
        const link = screen.getByRole('link', { name: /Open in Thoth/ })
        expect(link).toHaveAttribute('href', 'http://127.0.0.1:8333/notes/inbox/quick-thought.md')
        // The draft was cleared and the badge refreshed after the save.
        await waitFor(() => expect(storage.data['thoth:draft']).toBeUndefined())
    })

    it('switching to the bookmark tab reveals bookmark fields and posts the bookmark line', async () => {
        stubServer({
            '/api/v1/health': { status: 'ok' },
            '/api/v1/settings': settingsBody,
            '/api/v1/capture/check': { exists: false },
            '/api/v1/capture': { path: 'links/bookmarks.md', title: 'Go Channels', type: 'bookmark' },
        })
        renderPopup()
        await screen.findByLabelText('Text')
        fireEvent.click(screen.getByText('Bookmark'))

        await userEvent.type(screen.getByLabelText('Title'), 'Go Channels')
        await userEvent.type(screen.getByLabelText('URL'), 'https://go.dev/blog/channels')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))

        expect(await screen.findByText('Saved to links/bookmarks.md')).toBeInTheDocument()
        const postCall = vi
            .mocked(fetch)
            .mock.calls.find(([u]) => (u as string) === 'http://127.0.0.1:8333/api/v1/capture')
        const body = JSON.parse((postCall?.[1] as { body?: string } | undefined)?.body ?? '{}') as {
            kind: string
            url: string
        }
        expect(body.kind).toBe('bookmark')
        expect(body.url).toBe('https://go.dev/blog/channels')
    })

    it('blocks a duplicate bookmark with the "already saved → open it" state', async () => {
        stubServer({
            '/api/v1/health': { status: 'ok' },
            '/api/v1/settings': settingsBody,
            '/api/v1/capture/check': { exists: true, path: 'links/bookmarks.md' },
        })
        renderPopup()
        await screen.findByLabelText('Text')
        fireEvent.click(screen.getByText('Bookmark'))
        await userEvent.type(screen.getByLabelText('URL'), 'https://go.dev/blog/channels')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))

        expect(await screen.findByText(/Already saved/)).toBeInTheDocument()
        const link = screen.getByRole('link', { name: /Open it/ })
        expect(link).toHaveAttribute('href', 'http://127.0.0.1:8333/notes/links/bookmarks.md')
    })

    it('shows a validation error instead of saving an empty note', async () => {
        stubServer({ '/api/v1/health': { status: 'ok' }, '/api/v1/settings': settingsBody })
        renderPopup()
        await screen.findByLabelText('Text')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))
        expect(await screen.findByText('Text is required')).toBeInTheDocument()
    })

    it('saves a selection draft with its derived title and domain tag', async () => {
        stubServer({
            '/api/v1/health': { status: 'ok' },
            '/api/v1/settings': settingsBody,
            '/api/v1/capture': { path: 'inbox/welcome.md', title: 'Welcome to the Turborepo documentation!', type: 'inbox' },
        })
        const storage = memoryStorage({
            'thoth:draft': JSON.stringify({
                kind: 'selection',
                url: 'https://turborepo.dev/docs',
                title: 'Welcome to the Turborepo documentation!',
                text: 'Welcome to the Turborepo documentation! What is Turborepo?',
                tags: ['turborepo'],
            }),
        })
        renderPopup(storage)
        await screen.findByLabelText('Text')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))
        expect(await screen.findByText('Saved to inbox/welcome.md')).toBeInTheDocument()
        const postCall = vi
            .mocked(fetch)
            .mock.calls.find(([u]) => (u as string) === 'http://127.0.0.1:8333/api/v1/capture')
        const body = JSON.parse((postCall?.[1] as { body?: string } | undefined)?.body ?? '{}') as {
            title: string
            tags: string[]
        }
        expect(body.title).toBe('Welcome to the Turborepo documentation!')
        expect(body.tags).toEqual(['turborepo'])
    })

    it('defaults a bookmark save to the stored last-used category', async () => {
        stubServer({
            '/api/v1/health': { status: 'ok' },
            '/api/v1/settings': settingsBody,
            '/api/v1/capture/check': { exists: false },
            '/api/v1/capture': { path: 'links/bookmarks.md', title: 'Go Docs', type: 'bookmark' },
        })
        const storage = memoryStorage({
            [BASE_URL_KEY]: 'http://127.0.0.1:8333',
            'thoth:lastCategory': 'docs',
            'thoth:draft': JSON.stringify({ kind: 'bookmark', url: 'https://go.dev/doc', title: 'Go Docs' }),
        })
        renderPopup(storage)
        await screen.findByLabelText('URL')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))
        expect(await screen.findByText('Saved to links/bookmarks.md')).toBeInTheDocument()
        const postCall = vi
            .mocked(fetch)
            .mock.calls.find(([u]) => (u as string) === 'http://127.0.0.1:8333/api/v1/capture')
        const body = JSON.parse((postCall?.[1] as { body?: string } | undefined)?.body ?? '{}') as {
            category: string
        }
        expect(body.category).toBe('docs')
    })

    it('remembers the category used for a bookmark save', async () => {
        stubServer({
            '/api/v1/health': { status: 'ok' },
            '/api/v1/settings': settingsBody,
            '/api/v1/capture/check': { exists: false },
            '/api/v1/capture': { path: 'links/bookmarks.md', title: 'Go Docs', type: 'bookmark' },
        })
        const storage = memoryStorage({ [BASE_URL_KEY]: 'http://127.0.0.1:8333' })
        renderPopup(storage)
        await screen.findByLabelText('Text')
        fireEvent.click(screen.getByText('Bookmark'))
        await userEvent.type(screen.getByLabelText('URL'), 'https://go.dev/doc')
        fireEvent.click(screen.getByRole('button', { name: /^Save/ }))
        expect(await screen.findByText('Saved to links/bookmarks.md')).toBeInTheDocument()
        // The category the save used (the 'unfiled' default here) is remembered
        // for the next capture.
        await waitFor(() => expect(storage.data['thoth:lastCategory']).toBe('unfiled'))
    })

    it('does not silently fall back to a default port for a down custom URL', async () => {
        stubServer({}) // every route is down
        const { ext } = renderPopup()
        // Let the mount's auto-discovery settle before editing the URL.
        await screen.findByText(/Thoth is not running/)
        const input = screen.getByLabelText('Server URL')
        fireEvent.change(input, { target: { value: 'http://127.0.0.1:9999' } })
        fireEvent.click(screen.getByRole('button', { name: 'Connect' }))
        // The entered server is reported as down — not "Connected to 8333".
        expect(await screen.findByText(/That server is not running/)).toBeInTheDocument()
        expect(screen.queryByText(/Connected to/)).not.toBeInTheDocument()
        expect(ext.permissionRequests).toEqual([])
    })

    it('clears a stored custom URL when the host permission grant is denied', async () => {
        stubServer({})
        const fake = fakeBrowserAPI()
        fake.api.permissions!.request = async () => false
        const storage = memoryStorage({ [BASE_URL_KEY]: 'http://192.168.1.5:8333' })
        render(
            <ConfigProvider theme={popupTheme}>
                <PopupApp storage={storage} ext={fake.api} />
            </ConfigProvider>
        )
        // Mount settles: the stored custom URL is probed (down), discovery is
        // down too — "Thoth is not running".
        await screen.findByText(/Thoth is not running/)
        const input = screen.getByLabelText('Server URL')
        fireEvent.change(input, { target: { value: 'http://192.168.1.5:8333' } })
        fireEvent.click(screen.getByRole('button', { name: 'Connect' }))
        expect(await screen.findByText(/Permission to reach/)).toBeInTheDocument()
        // A denied host must not linger in storage.
        await waitFor(() => expect(storage.data[BASE_URL_KEY]).toBeUndefined())
    })
})
