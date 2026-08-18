import { memo } from 'react'
import { Avatar, Flex, Tooltip, theme } from 'antd'
import { RobotOutlined } from '@ant-design/icons'
import type { ChatMessage } from '../hooks/useChat'
import { CopyButton } from './CopyButton'
import { Markdown } from './Markdown'

// Memoized: message objects from the store are referentially stable, so
// only the row whose props actually changed re-renders during streaming.
export const MessageItem = memo(function MessageItem({
    message,
    streaming
}: {
    message: ChatMessage
    streaming?: boolean
}) {
    const isUser = message.role === 'user'

    return (
        <Flex align="flex-start" gap={10} justify={isUser ? 'flex-end' : 'flex-start'}>
            {!isUser && <AssistantIcon />}
            <div
                className={
                    isUser
                        ? 'group relative max-w-[80%] rounded-xl rounded-br-sm bg-accent px-4 py-2.5 text-accent-ink'
                        : 'group relative max-w-[85%] rounded-xl rounded-bl-sm border border-line bg-surface px-4 py-2.5'
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
                {isUser ? (
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
                ) : (
                    <Markdown
                        className="pr-6"
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

// AssistantIcon is the small avatar shown to the left of every assistant
// message, mirroring the app's accent color.
function AssistantIcon() {
    const { token } = theme.useToken()
    return (
        <Avatar
            aria-hidden="true"
            size={28}
            shape="square"
            icon={<RobotOutlined />}
            className="mt-0.5 shrink-0 shadow-sm"
            style={{
                borderRadius: token.borderRadius,
                backgroundColor: token.colorPrimary,
                color: token.colorTextLightSolid
            }}
        />
    )
}
