import { memo } from 'react'
import { Flex, Tooltip } from 'antd'
import type { ChatMessage } from '../../hooks/useChat'
import { AssistantIcon } from './AssistantIcon'
import { CopyButton } from '../../shared/CopyButton'
import { SaveAsNote } from './SaveAsNote'
import { Markdown } from '../../shared/Markdown'

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
                        ? 'group relative max-w-[80%] rounded-2xl rounded-br-md bg-accent px-4 py-2.5 text-accent-ink'
                        : 'group relative max-w-[85%] rounded-2xl rounded-bl-md border border-line bg-surface px-4 py-2.5 shadow-sm'
                }
            >
                {!isUser && !streaming && (
                    <Tooltip title="Copy message">
                        <span className="absolute right-2 top-2">
                            <CopyButton
                                text={message.content}
                                label="Copy message"
                                toast="Message copied to clipboard"
                            />
                        </span>
                    </Tooltip>
                )}
                {!isUser && !streaming && <SaveAsNote content={message.content} />}
                {isUser ? (
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
                ) : (
                    <Markdown
                        className="pr-6"
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
