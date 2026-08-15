import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CodeBlock } from './CodeBlock'

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

// Markdown renders GFM markdown with Shiki-highlighted code blocks. Prose
// styling comes from the caller's wrapper (prose classes), not here.
export function Markdown({ children }: { children: string }) {
    return (
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
            {children}
        </ReactMarkdown>
    )
}
