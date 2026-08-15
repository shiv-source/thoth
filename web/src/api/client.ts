import axios from 'axios'
import { z } from 'zod'

const SearchResult = z.object({ path: z.string(), title: z.string(), kind: z.string(), snippet: z.string() })
export type SearchResult = z.infer<typeof SearchResult>

const Note = z.object({ path: z.string(), content: z.string() })
export type Note = z.infer<typeof Note>

type TreeNodeShape = { name: string; path: string; is_dir: boolean; children: TreeNodeShape[] | null }

const TreeNodeSchema: z.ZodType<TreeNodeShape> = z.lazy(() =>
  z.object({ name: z.string(), path: z.string(), is_dir: z.boolean(), children: z.array(TreeNodeSchema).nullable() }),
)
export type TreeNode = z.infer<typeof TreeNodeSchema>

export const Settings = z.object({
  wiki_path: z.string(),
  repo_url: z.string(),
  sync_enabled: z.boolean(),
})
export type Settings = z.infer<typeof Settings>

export const GitHubIdentity = z.object({
  username: z.string(),
  display_name: z.string(),
  email: z.string(),
  avatar_url: z.string(),
  profile_url: z.string(),
  scopes: z.string(),
  account_created_at: z.string(),
  account_updated_at: z.string(),
})
export type GitHubIdentity = z.infer<typeof GitHubIdentity>

export const GitHubRepo = z.object({ full_name: z.string(), clone_url: z.string(), private: z.boolean(), description: z.string() })
export type GitHubRepo = z.infer<typeof GitHubRepo>

const Conversation = z.object({ id: z.string(), title: z.string(), created_at: z.string() })
export type Conversation = z.infer<typeof Conversation>

const Message = z.object({
  id: z.number(),
  conversation_id: z.string(),
  role: z.enum(['user', 'assistant']),
  content: z.string(),
  created_at: z.string(),
})
export type Message = z.infer<typeof Message>

export const Health = z.object({
  status: z.string(),
  claude: z.object({ found: z.boolean(), path: z.string() }),
  wiki: z.object({ path: z.string(), exists: z.boolean() }),
  version: z.string(),
})
export type Health = z.infer<typeof Health>

export const DoctorCheck = z.object({ name: z.string(), ok: z.boolean(), message: z.string() })
export type DoctorCheck = z.infer<typeof DoctorCheck>

// The single axios instance every request goes through.
const http = axios.create({ timeout: 10000 })

async function get<T>(url: string, schema: z.ZodType<T>): Promise<T> {
  const res = await http.get(url)
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
  search: (q: string) => get(`/api/search?q=${encodeURIComponent(q)}`, z.object({ results: z.array(SearchResult) })),
  note: (path: string) => get(`/api/notes?path=${encodeURIComponent(path)}`, Note),
  tree: () => get('/api/wiki/tree', z.object({ nodes: z.array(TreeNodeSchema) })),
  settings: () => get('/api/settings', Settings),
  saveSettings: async (s: Settings): Promise<Settings> => {
    const res = await http.put('/api/settings', s)
    return parseBody(res, Settings)
  },
  health: () => get('/api/health', Health),
  doctor: () => get('/api/doctor', z.object({ checks: z.array(DoctorCheck) })),
  listConversations: () => get('/api/conversations', z.object({ conversations: z.array(Conversation) })),
  deleteConversation: async (id: string): Promise<void> => {
    await http.delete(`/api/conversations/${encodeURIComponent(id)}`)
  },
  getConversation: (id: string) => get(`/api/conversations/${encodeURIComponent(id)}`, z.object({ conversation: Conversation, messages: z.array(Message) })),
  gitSetup: async (url: string): Promise<{ ok: boolean; error?: string }> => {
    const res = await http.post('/api/git/setup', { url })
    return parseBody(res, z.object({ ok: z.boolean(), error: z.string().optional() }))
  },
  githubAuth: () => get('/api/github/auth', GitHubIdentity),
  githubRepos: () => get('/api/github/repos', z.object({ repos: z.array(GitHubRepo) })),
  connectGitHub: async (token: string): Promise<GitHubIdentity> => {
    try {
      const res = await http.post('/api/github/auth', { token })
      return parseBody(res, GitHubIdentity)
    } catch (err) {
      throw new Error(axiosErrorMessage(err), { cause: err })
    }
  },
  disconnectGitHub: async (): Promise<void> => {
    try {
      await http.delete('/api/github/auth')
    } catch (err) {
      throw new Error(axiosErrorMessage(err), { cause: err })
    }
  },
}
