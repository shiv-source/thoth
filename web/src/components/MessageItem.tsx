import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { ChatMessage } from '../hooks/useChat'

export function MessageItem({ message, streaming }: { message: ChatMessage; streaming?: boolean }) {
  const isUser = message.role === 'user'
  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'}`}>
      <div
        className={
          isUser
            ? 'max-w-[80%] rounded-2xl rounded-br-sm bg-ink-900 px-4 py-2.5 text-paper-100 dark:bg-paper-100 dark:text-ink-900'
            : 'max-w-[85%] rounded-2xl rounded-bl-sm border border-paper-200 bg-paper-100 px-4 py-2.5 dark:border-night-700 dark:bg-night-900'
        }
      >
        {isUser ? (
          <p className="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
        ) : (
          <div className="prose-sm max-w-none prose-headings:font-display prose-headings:text-accent-700 dark:prose-invert dark:prose-headings:text-accent-500">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
            {streaming && <span className="ml-0.5 inline-block h-4 w-1.5 animate-pulse rounded bg-accent-600 align-text-bottom" />}
          </div>
        )}
      </div>
    </div>
  )
}
