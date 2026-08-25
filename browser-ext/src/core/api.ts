import { probeServer } from './server'
import type { CaptureInput, CaptureResponse, DuplicateCheck } from './types'

// ThothError carries the server's {"error":...} body plus, for 409 (URL
// already saved), the existing file path so the UI can offer "open it".
export class ThothError extends Error {
    readonly status?: number
    readonly path?: string

    constructor(message: string, status?: number, path?: string) {
        super(message)
        this.name = 'ThothError'
        this.status = status
        this.path = path
    }
}

// ThothClient talks to the Thoth REST API over plain fetch. MV3
// host_permissions lift CORS, so the client needs no server cooperation
// beyond the endpoints themselves.
export class ThothClient {
    constructor(private readonly baseUrl: string) {}

    async health(): Promise<boolean> {
        return probeServer(this.baseUrl)
    }

    async capture(input: CaptureInput): Promise<CaptureResponse> {
        return this.request<CaptureResponse>('/api/v1/capture', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        })
    }

    async checkDuplicate(url: string): Promise<DuplicateCheck> {
        return this.request<DuplicateCheck>(`/api/v1/capture/check?url=${encodeURIComponent(url)}`)
    }

    async inboxCount(): Promise<number> {
        const body = await this.request<{ count: number }>('/api/v1/capture/inbox-count')
        return body.count
    }

    async summarize(input: { url: string; title: string; text: string }): Promise<CaptureResponse> {
        return this.request<CaptureResponse>('/api/v1/capture/summarize', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        })
    }

    async folders(): Promise<string[]> {
        const body = await this.request<{ wiki_folders?: string[] }>('/api/v1/settings')
        return body.wiki_folders ?? []
    }

    private async request<T>(path: string, init?: RequestInit): Promise<T> {
        let res: Response
        try {
            res = await fetch(this.baseUrl + path, init)
        } catch {
            throw new ThothError('server not reachable')
        }
        if (!res.ok) {
            const body = (await res.json().catch(() => null)) as { error?: string; path?: string } | null
            if (res.status === 409) {
                throw new ThothError(body?.error ?? 'url already saved', 409, body?.path)
            }
            throw new ThothError(body?.error ?? `request failed (${res.status})`, res.status)
        }
        return (await res.json()) as T
    }
}
