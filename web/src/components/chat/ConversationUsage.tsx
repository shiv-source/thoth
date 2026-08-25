import { Fragment } from 'react'
import type { TokenUsage } from '../../ws/protocol'
import { UsageTokens } from './UsageTokens'

// ConversationUsage renders the whole conversation's accumulated token usage —
// total input/output across every assistant turn, plus cache counters when
// present — as a muted line beside the composer's model chip. It renders
// nothing when no turn reported usage.
export function ConversationUsage({ usage }: { usage: TokenUsage }) {
    if (
        usage.input_tokens === 0 &&
        usage.output_tokens === 0 &&
        usage.cache_read_tokens === 0 &&
        usage.cache_write_tokens === 0
    ) {
        return null
    }
    return (
        <span className="inline-flex items-center gap-1.5 text-xs text-subtle" aria-label="Conversation token usage">
            <UsageTokens input={usage.input_tokens} output={usage.output_tokens} />
            {usage.cache_read_tokens > 0 && (
                <Fragment>
                    <span aria-hidden="true">·</span>
                    <span>{usage.cache_read_tokens.toLocaleString()} cache read</span>
                </Fragment>
            )}
            {usage.cache_write_tokens > 0 && (
                <Fragment>
                    <span aria-hidden="true">·</span>
                    <span>{usage.cache_write_tokens.toLocaleString()} cache write</span>
                </Fragment>
            )}
        </span>
    )
}
