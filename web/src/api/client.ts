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
  host: z.string(),
  port: z.number(),
  claude_bin: z.string(),
  permission_mode: z.string(),
  model: z.string(),
})
export type Settings = z.infer<typeof Settings>

const Conversation = z.object({ id: z.string(), title: z.string(), created_at: z.string() })
export type Conversation = z.infer<typeof Conversation>

async function get<T>(url: string, schema: z.ZodType<T>): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return schema.parse(await res.json())
}

export const api = {
  search: (q: string) => get(`/api/search?q=${encodeURIComponent(q)}`, z.object({ results: z.array(SearchResult) })),
  note: (path: string) => get(`/api/notes?path=${encodeURIComponent(path)}`, Note),
  tree: () => get('/api/wiki/tree', z.object({ nodes: z.array(TreeNodeSchema) })),
  settings: () => get('/api/settings', Settings),
  saveSettings: async (s: Settings): Promise<Settings> => {
    const res = await fetch('/api/settings', { method: 'PUT', headers: { 'content-type': 'application/json' }, body: JSON.stringify(s) })
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
    return Settings.parse(await res.json())
  },
  listConversations: () => get('/api/conversations', z.object({ conversations: z.array(Conversation) })),
}
