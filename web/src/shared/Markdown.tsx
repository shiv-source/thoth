import type { ReactNode } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CodeBlock } from './CodeBlock'

// PROSE is the shared typography wrapper for every markdown surface
// (light theme only).
const PROSE =
    'prose prose-sm max-w-none prose-headings:font-display prose-headings:text-heading prose-code:before:content-none prose-code:after:content-none prose-pre:rounded-lg'

// Fenced code blocks route through Shiki (CodeBlock); inline code keeps the
// default prose styling. The pre wrapper is unwrapped so the highlight's own
// pre takes its place instead of nesting two.
const components: Components = {
    pre({ children }) {
        return <>{children}</>
    },
    code({ className, children }) {
        const match = /language-([\w-]+)/.exec(className ?? '')
        const code = typeof children === 'string' ? children.replace(/\n$/, '') : ''
        if (match) return <CodeBlock code={code} lang={match[1] ?? 'text'} />
        return <code className={className}>{children}</code>
    }
}

// Markdown renders GFM markdown with Shiki-highlighted code blocks inside the
// shared prose wrapper. `trailing` renders after the content (e.g. the
// streaming caret in chat); `className` is appended to the wrapper.
export function Markdown({
    children,
    trailing,
    className = ''
}: {
    children: string
    trailing?: ReactNode
    className?: string
}) {
    return (
        <div className={`${PROSE} ${className}`}>
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
                {children}
            </ReactMarkdown>
            {trailing}
        </div>
    )
}
