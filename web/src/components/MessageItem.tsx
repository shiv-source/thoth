import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { ChatMessage } from '../hooks/useChat'

export function MessageItem({ message, streaming }: { message: ChatMessage; streaming?: boolean }) {
  const isUser = message.role === 'user'
  return (
    <div className={`flex items-start gap-2.5 ${isUser ? 'justify-end' : 'justify-start'}`}>
      {!isUser && <AssistantIcon />}
      <div
        className={
          isUser
            ? 'max-w-[80%] rounded-xl rounded-br-sm bg-accent px-4 py-2.5 text-accent-ink'
            : 'max-w-[85%] rounded-xl rounded-bl-sm border border-line bg-surface px-4 py-2.5'
        }
      >
        {isUser ? (
          <p className="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
        ) : (
          <div className="prose prose-sm max-w-none prose-headings:font-display prose-headings:text-heading prose-code:before:content-none prose-code:after:content-none prose-pre:rounded-lg dark:prose-invert">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
            {streaming && <span className="ml-0.5 inline-block h-4 w-1.5 animate-pulse rounded bg-accent align-text-bottom" />}
          </div>
        )}
      </div>
    </div>
  )
}

// AssistantIcon is the small avatar shown to the left of every assistant
// message, mirroring the app's accent color.
function AssistantIcon() {
  return (
    <span aria-hidden="true"
      className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-accent text-accent-ink">
      <svg viewBox="0 0 16 16" fill="currentColor" className="h-4 w-4">
        <path d="M8 0l1.9 4.1L14 6l-4.1 1.9L8 12l-1.9-4.1L2 6l4.1-1.9L8 0z" />
        <path d="M12.5 10l.9 2 2 .9-2 .9-.9 2-.9-2-2-.9 2-.9.9-2z" opacity="0.7" />
      </svg>
    </span>
  )
}
