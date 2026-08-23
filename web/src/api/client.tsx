import axios from 'axios'
import { z } from 'zod'

const SearchResult = z.object({ path: z.string(), title: z.string(), kind: z.string(), snippet: z.string() })
export type SearchResult = z.infer<typeof SearchResult>

const Note = z.object({ path: z.string(), content: z.string() })
export type Note = z.infer<typeof Note>

// SaveNoteResponse is the promotion endpoint's reply: the created note's
// wiki-relative path plus the derived title/type (advisory — the file holds
// the real frontmatter).
const SaveNoteResponse = z.object({ path: z.string(), title: z.string(), type: z.string() })
export type SaveNoteResponse = z.infer<typeof SaveNoteResponse>

type TreeNodeShape = {
    name: string
    path: string
    is_dir: boolean
    children: TreeNodeShape[] | null
    error?: string
}

const TreeNodeSchema: z.ZodType<TreeNodeShape> = z.lazy(() =>
    z.object({
        name: z.string(),
        path: z.string(),
        is_dir: z.boolean(),
        children: z.array(TreeNodeSchema).nullable(),
        error: z.string().optional()
    })
)
export type TreeNode = z.infer<typeof TreeNodeSchema>

// Provider is one row of the providers table: name, its base URL override and
// key presence (the key itself is write-only — GET reports has_api_key only)
// plus the number of models registered under it. ProviderInput is the
// create/update body.
export const Provider = z.object({
    id: z.number(),
    name: z.string(),
    base_url: z.string(),
    has_api_key: z.boolean(),
    model_count: z.number()
})
export type Provider = z.infer<typeof Provider>
export type ProviderInput = { name: string; base_url?: string; api_key?: string }

export const Settings = z.object({
    wiki_path: z.string(),
    wiki_folders: z.array(z.string()),
    model: z.string(),
    repo_url: z.string(),
    sync_enabled: z.boolean(),
    context_injection: z.boolean()
})
export type Settings = z.infer<typeof Settings>

// LLMModel is one row of the llm_models table (provider is the owning
// providers row's name via a join; provider_id is its id, 0 for the
// Unassigned catch-all). ModelInput is the create/update body.
export const LLMModel = z.object({
    id: z.number(),
    value: z.string(),
    name: z.string(),
    tag: z.string(),
    provider: z.string(),
    provider_id: z.number()
})
export type LLMModel = z.infer<typeof LLMModel>
export const ModelGroup = z.object({ provider: z.string(), models: z.array(LLMModel) })
export type ModelGroup = z.infer<typeof ModelGroup>
export type ModelInput = { value: string; name: string; tag?: string; provider_id?: number | null }

export const GitHubIdentity = z.object({
    username: z.string(),
    display_name: z.string(),
    email: z.string(),
    avatar_url: z.string(),
    profile_url: z.string(),
    scopes: z.string(),
    account_created_at: z.string(),
    account_updated_at: z.string()
})
export type GitHubIdentity = z.infer<typeof GitHubIdentity>

export const GitHubRepo = z.object({
    full_name: z.string(),
    clone_url: z.string(),
    private: z.boolean(),
    description: z.string()
})
export type GitHubRepo = z.infer<typeof GitHubRepo>

const Conversation = z.object({ id: z.string(), title: z.string(), created_at: z.string() })
export type Conversation = z.infer<typeof Conversation>

const TokenUsage = z.object({
    input_tokens: z.number(),
    output_tokens: z.number(),
    cache_read_tokens: z.number(),
    cache_write_tokens: z.number()
})
export type TokenUsage = z.infer<typeof TokenUsage>

const Message = z.object({
    id: z.number(),
    conversation_id: z.string(),
    role: z.enum(['user', 'assistant']),
    content: z.string(),
    created_at: z.string(),
    usage: TokenUsage.optional()
})
export type Message = z.infer<typeof Message>

export const Health = z.object({
    status: z.string(),
    backend: z.object({
        name: z.string(),
        api_key_configured: z.boolean(),
        model: z.string(),
        provider: z.string()
    }),
    wiki: z.object({ path: z.string(), exists: z.boolean() }),
    version: z.string(),
    dev: z.boolean(),
    commit: z.string(),
    default_wiki_path: z.string()
})
export type Health = z.infer<typeof Health>

export const DoctorCheck = z.object({ name: z.string(), ok: z.boolean(), message: z.string() })
export type DoctorCheck = z.infer<typeof DoctorCheck>

// The single axios instance every request goes through.
const http = axios.create({ timeout: 10000 })

async function get<T>(url: string, schema: z.ZodType<T>, signal?: AbortSignal): Promise<T> {
    // Only attach the config when a signal exists: every non-search caller
    // keeps the bare-url call shape (and its mock assertions).
    const res = signal ? await http.get(url, { signal }) : await http.get(url)
    return schema.parse(res.data)
}

function parseBody<T>(res: { data: unknown }, schema: z.ZodType<T>): T {
    return schema.parse(res.data)
}

/** Reads the server's {"error":"<msg>"} body, falling back to the status. */
function axiosErrorMessage(err: unknown): string {
    if (axios.isAxiosError(err)) {
        const body = err.response?.data as { error?: unknown } | undefined
        if (typeof body?.error === 'string') return body.error
        if (err.response) return `${err.response.status} ${err.response.statusText}`
    }
    return 'request failed'
}

export const api = {
    search: (q: string, signal?: AbortSignal) =>
        get(`/api/v1/search?q=${encodeURIComponent(q)}`, z.object({ results: z.array(SearchResult) }), signal),
    note: (path: string) => get(`/api/v1/notes?path=${encodeURIComponent(path)}`, Note),
    // saveNote promotes content into a permanent wiki note. folder is
    // optional: the server defaults to the first configured folder.
    saveNote: async (input: { content: string; folder?: string }): Promise<SaveNoteResponse> => {
        const res = await http.post('/api/v1/notes', input)
        return parseBody(res, SaveNoteResponse)
    },
    tree: () => get('/api/v1/wiki/tree', z.object({ nodes: z.array(TreeNodeSchema) })),
    settings: () => get('/api/v1/settings', Settings),
    listDirs: (path: string) =>
        get(`/api/v1/fs/dirs?path=${encodeURIComponent(path)}`, z.object({ dirs: z.array(z.string()) })),
    models: () => get('/api/v1/models', z.object({ groups: z.array(ModelGroup) })),
    createModel: async (input: ModelInput): Promise<LLMModel> => {
        const res = await http.post('/api/v1/models', input)
        return parseBody(res, LLMModel)
    },
    updateModel: async (id: number, input: ModelInput): Promise<LLMModel> => {
        const res = await http.put(`/api/v1/models/${id}`, input)
        return parseBody(res, LLMModel)
    },
    deleteModel: async (id: number): Promise<void> => {
        await http.delete(`/api/v1/models/${id}`)
    },
    providers: () => get('/api/v1/providers', z.object({ providers: z.array(Provider) })),
    createProvider: async (input: ProviderInput): Promise<Provider> => {
        const res = await http.post('/api/v1/providers', input)
        return parseBody(res, Provider)
    },
    updateProvider: async (id: number, input: ProviderInput): Promise<Provider> => {
        const res = await http.put(`/api/v1/providers/${id}`, input)
        return parseBody(res, Provider)
    },
    deleteProvider: async (id: number): Promise<void> => {
        await http.delete(`/api/v1/providers/${id}`)
    },
    saveSettings: async (s: Settings): Promise<Settings> => {
        const res = await http.put('/api/v1/settings', s)
        return parseBody(res, Settings)
    },
    // exportWiki downloads the wiki as a zip. includeHistory pulls in dotfiles
    // (.git) so git history travels. The blob is handed to the browser as a
    // download named thoth-wiki-YYYY-MM-DD.zip.
    exportWiki: async (includeHistory = false): Promise<void> => {
        const res = await http.get('/api/v1/wiki/export', {
            responseType: 'blob',
            params: includeHistory ? { history: '1' } : undefined
        })
        const url = URL.createObjectURL(res.data as Blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `thoth-wiki-${new Date().toISOString().slice(0, 10)}.zip`
        a.click()
        URL.revokeObjectURL(url)
    },
    // importWiki uploads a wiki zip and returns how many files were merged and
    // the pre-import backup directory (null when the wiki was empty). onUpload
    // reports upload progress as a 0-100 percentage.
    importWiki: async (
        file: File,
        onUpload?: (percent: number) => void
    ): Promise<{ files: number; backup: string | null }> => {
        const form = new FormData()
        form.append('file', file)
        try {
            const res = await http.post('/api/v1/wiki/import', form, {
                onUploadProgress: (e) => {
                    if (onUpload && e.total) onUpload(Math.round((e.loaded / e.total) * 100))
                }
            })
            return parseBody(res, z.object({ files: z.number(), backup: z.string().nullable() }))
        } catch (err) {
            throw new Error(axiosErrorMessage(err), { cause: err })
        }
    },
    health: () => get('/api/v1/health', Health),
    doctor: () => get('/api/v1/doctor', z.object({ checks: z.array(DoctorCheck) })),
    listConversations: () => get('/api/v1/conversations', z.object({ conversations: z.array(Conversation) })),
    deleteConversation: async (id: string): Promise<void> => {
        await http.delete(`/api/v1/conversations/${encodeURIComponent(id)}`)
    },
    getConversation: (id: string) =>
        get(
            `/api/v1/conversations/${encodeURIComponent(id)}`,
            z.object({ conversation: Conversation, messages: z.array(Message) })
        ),
    gitSetup: async (url: string): Promise<{ ok: boolean; error?: string }> => {
        const res = await http.post('/api/v1/git/setup', { url })
        return parseBody(res, z.object({ ok: z.boolean(), error: z.string().optional() }))
    },
    githubAuth: () => get('/api/v1/github/auth', GitHubIdentity),
    githubRepos: () => get('/api/v1/github/repos', z.object({ repos: z.array(GitHubRepo) })),
    connectGitHub: async (token: string): Promise<GitHubIdentity> => {
        try {
            const res = await http.post('/api/v1/github/auth', { token })
            return parseBody(res, GitHubIdentity)
        } catch (err) {
            throw new Error(axiosErrorMessage(err), { cause: err })
        }
    },
    disconnectGitHub: async (): Promise<void> => {
        try {
            await http.delete('/api/v1/github/auth')
        } catch (err) {
            throw new Error(axiosErrorMessage(err), { cause: err })
        }
    }
}
