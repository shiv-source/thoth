// The "dev server" the extension e2e runs against: a small Node HTTP server
// implementing the capture API contract (health, settings, capture, check,
// inbox-count, summarize, read-later). It records every capture POST so the
// test can assert what the popup actually sent, and exposes test-control
// endpoints for resetting state and registering "already saved" URLs for the
// dedup flow. The real Go server's wiki-write behavior is covered by the Go
// suite; this mock isolates the extension's end-to-end browser behavior.
import { createServer } from 'node:http'

const PORT = Number(process.env.PORT ?? 8337)
const saved = new Set()
const captures = []
const readLater = [
    { title: 'Long read', url: 'https://example.com/long', reason: 'weekend' },
    { title: 'Go modules', url: 'https://go.dev/doc/modules', reason: 'go' }
]

function kebab(input) {
    return (
        String(input ?? 'capture')
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '') || 'capture'
    )
}

function captureResponse(body) {
    const url = body.url ?? ''
    const title = body.title || url || 'capture'
    switch (body.kind) {
        case 'bookmark':
            return { path: 'links/bookmarks.md', title, type: 'bookmark' }
        case 'readlater':
            return { path: 'links/read-later.md', title, type: 'read-later' }
        case 'selection':
        case 'note': {
            const folder = body.folder ?? 'inbox'
            const base = kebab(body.title || (body.text || '').split('\n')[0])
            return { path: `${folder}/${base}.md`, title: body.title || base, type: folder }
        }
        default:
            return { path: 'inbox/capture.md', title, type: 'inbox' }
    }
}

function readJson(req) {
    return new Promise((resolve) => {
        let data = ''
        req.on('data', (chunk) => {
            data += chunk
        })
        req.on('end', () => {
            try {
                resolve(JSON.parse(data || '{}'))
            } catch {
                resolve({})
            }
        })
    })
}

const server = createServer(async (req, res) => {
    const url = new URL(req.url ?? '/', `http://127.0.0.1:${PORT}`)
    const send = (code, body) => {
        res.writeHead(code, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify(body))
    }

    // Test-control endpoints — not part of the capture contract.
    if (url.pathname === '/__captures' && req.method === 'GET') return send(200, { captures })
    if (url.pathname === '/__captures' && req.method === 'DELETE') {
        captures.length = 0
        return send(200, { ok: true })
    }
    if (url.pathname === '/__saved' && req.method === 'POST') {
        const body = await readJson(req)
        if (body.url) saved.add(body.url)
        return send(200, { ok: true })
    }

    // Capture API contract.
    if (url.pathname === '/api/v1/health') return send(200, { status: 'ok', version: 'e2e-mock' })
    if (url.pathname === '/api/v1/settings') {
        return send(200, { wiki_folders: ['inbox', 'knowledge', 'links', 'projects', 'setup', 'todos', 'daily'] })
    }
    if (url.pathname === '/api/v1/capture/inbox-count') return send(200, { count: 0 })

    if (url.pathname === '/api/v1/capture/check') {
        const exists = saved.has(url.searchParams.get('url') ?? '')
        return send(200, exists ? { exists: true, path: 'links/bookmarks.md' } : { exists: false })
    }

    if (url.pathname === '/api/v1/capture' && req.method === 'POST') {
        const body = await readJson(req)
        captures.push(body)
        return send(201, captureResponse(body))
    }

    if (url.pathname === '/api/v1/capture/summarize' && req.method === 'POST') {
        const body = await readJson(req)
        captures.push({ kind: 'summarize', ...body })
        return send(201, {
            path: `knowledge/summary-${kebab(body.title)}.md`,
            title: `Summary: ${body.title ?? 'page'}`,
            type: 'knowledge'
        })
    }

    if (url.pathname === '/api/v1/capture/read-later' && req.method === 'GET') return send(200, { items: readLater })
    if (url.pathname === '/api/v1/capture/read-later' && req.method === 'DELETE') {
        const target = url.searchParams.get('url')
        for (let i = readLater.length - 1; i >= 0; i--) {
            if (readLater[i].url === target) readLater.splice(i, 1)
        }
        return send(200, { ok: true })
    }

    return send(404, { error: 'not found' })
})

server.listen(PORT, '127.0.0.1', () => {
    console.log(`mock thoth capture server on http://127.0.0.1:${PORT}`)
})
