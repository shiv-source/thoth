import type { TokenUsage } from '../ws/chat'

// UsageLine renders a completed turn's token breakdown as a muted footer line
// under the final assistant message. It renders nothing when usage is absent
// (provider reported none, or an older server sent no usage field).
export function UsageLine({ usage }: { usage: TokenUsage | null }) {
    if (usage === null) return null
    if (
        usage.input_tokens === 0 &&
        usage.output_tokens === 0 &&
        usage.cache_read_tokens === 0 &&
        usage.cache_write_tokens === 0
    ) {
        return null
    }
    const parts = [`${usage.input_tokens} in`, `${usage.output_tokens} out`]
    if (usage.cache_read_tokens > 0) parts.push(`${usage.cache_read_tokens} cache read`)
    if (usage.cache_write_tokens > 0) parts.push(`${usage.cache_write_tokens} cache write`)
    return (
        <span className="pl-0.5 text-xs text-subtle" aria-label="Token usage">
            {parts.join(' · ')}
        </span>
    )
}
