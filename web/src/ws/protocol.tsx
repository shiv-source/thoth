import { z } from 'zod'
import { ServerEvent } from './events'

// The WS frame shapes, validated at the socket seam exactly like the REST
// client validates responses (api/client.ts). serverMessageSchema is the
// single source of the ServerMessage type — a frame that parses as JSON but
// fails this schema is dropped, so garbage never reaches the UI. The server
// contract is internal/api/chat.go ↔ docs/api.md.

const tokenUsageSchema = z.object({
    input_tokens: z.number(),
    output_tokens: z.number(),
    cache_read_tokens: z.number(),
    cache_write_tokens: z.number()
})

// wiki_changed is the server push for wiki filesystem changes: the watcher
// batches one frame per debounce flush (changes may be empty on startup).
const wikiChangeOpSchema = z.enum(['create', 'write', 'remove', 'rename'])

const wikiChangeSchema = z.object({
    op: wikiChangeOpSchema,
    path: z.string()
})

export const serverMessageSchema = z.discriminatedUnion('type', [
    z.object({ type: z.literal(ServerEvent.AssistantStart) }),
    z.object({ type: z.literal(ServerEvent.AssistantThinking), text: z.string() }),
    z.object({ type: z.literal(ServerEvent.AssistantDelta), text: z.string() }),
    z.object({ type: z.literal(ServerEvent.ToolActivity), tool: z.string(), detail: z.string() }),
    z.object({
        type: z.literal(ServerEvent.TurnDone),
        conversation_id: z.string().optional(),
        usage: tokenUsageSchema.optional()
    }),
    z.object({
        type: z.literal(ServerEvent.WikiChanged),
        changes: z.array(wikiChangeSchema).optional()
    }),
    // sync_result is the auto-sync notification frame: a scheduled push
    // completed, so the UI can surface a toast without polling.
    z.object({
        type: z.literal(ServerEvent.SyncResult),
        sync_result: z.object({
            connection_id: z.number(),
            name: z.string(),
            ok: z.boolean(),
            error: z.string().optional()
        })
    }),
    z.object({ type: z.literal(ServerEvent.Error), message: z.string() })
])

export type ServerMessage = z.infer<typeof serverMessageSchema>

// TokenUsage is the per-turn token breakdown on turn_done. It is optional:
// a provider that reports no usage (or an older server) omits the field.
export type TokenUsage = z.infer<typeof tokenUsageSchema>

export type WikiChangeOp = z.infer<typeof wikiChangeOpSchema>

export type WikiChange = z.infer<typeof wikiChangeSchema>
