import { memo } from 'react'
import { Flex, Tooltip } from 'antd'
import type { ChatMessage } from '../../hooks/useChat'
import { AssistantIcon } from './AssistantIcon'
import { CopyButton } from '../../shared/CopyButton'
import { SaveAsNote } from './SaveAsNote'
import { Markdown } from '../../shared/Markdown'
import { UsageLine } from './UsageLine'

// Memoized: message objects from the store are referentially stable, so
// only the row whose props actually changed re-renders during streaming.
export const MessageItem = memo(function MessageItem({
    message,
    streaming,
    onOpenNote
}: {
    message: ChatMessage
    streaming?: boolean
    onOpenNote?: (path: string) => void
}) {
    const isUser = message.role === 'user'

    return (
        <Flex align="flex-start" gap={10} justify={isUser ? 'flex-end' : 'flex-start'}>
            {!isUser && <AssistantIcon />}
            <div
                className={
                    isUser
                        ? 'max-w-[80%] rounded-2xl rounded-br-md bg-accent px-4 py-2.5 text-accent-ink'
                        : 'max-w-[85%] rounded-2xl rounded-bl-md border border-line bg-surface px-4 py-2.5 shadow-sm'
                }
            >
                {!isUser && !streaming && (
                    <div className="mb-1.5 flex items-center justify-between gap-2 border-b border-line pb-1">
                        <span className="flex items-center gap-2">
                            {message.durationSecs !== undefined && (
                                <span className="text-xs tabular-nums text-subtle" aria-label="Turn duration">
                                    {formatDuration(message.durationSecs)}
                                </span>
                            )}
                            <UsageLine usage={message.usage ?? null} />
                        </span>
                        <div className="flex items-center">
                            <SaveAsNote content={message.content} />
                            <Tooltip title="Copy message">
                                <CopyButton
                                    text={message.content}
                                    label="Copy message"
                                    toast="Message copied to clipboard"
                                />
                            </Tooltip>
                        </div>
                    </div>
                )}
                {isUser ? (
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
                ) : (
                    <Markdown
                        onOpenNote={onOpenNote}
                        trailing={
                            streaming ? (
                                <span className="ml-0.5 inline-block h-4 w-1.5 animate-pulse rounded bg-accent align-text-bottom" />
                            ) : undefined
                        }
                    >
                        {message.content}
                    </Markdown>
                )}
            </div>
        </Flex>
    )
})

// formatDuration shows seconds with at most two decimals, trimming trailing
// zeros (12.5s, 12.34s). Full precision stays in the store and the database.
function formatDuration(secs: number): string {
    return `${Number(secs.toFixed(2))}s`
}
