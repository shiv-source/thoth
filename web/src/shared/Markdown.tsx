import type { ReactNode } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CodeBlock } from './CodeBlock'

// PROSE is the shared typography wrapper for every markdown surface
// (light theme only).
const PROSE =
    'prose prose-sm max-w-none prose-headings:font-display prose-headings:text-heading prose-code:before:content-none prose-code:after:content-none prose-pre:rounded-lg'

// A cited wiki note path: `*.md` with an optional directory prefix and no
// spaces (kebab-case names per the wiki contract). Inline code matching this
// renders as a clickable chip when the consumer provides onOpenNote; the
// character class keeps spaces, backticks and HTML out of chip territory, and
// a trailing newline (fenced blocks) fails the match.
const NOTE_PATH = /^[\w./-]+\.md$/

// NoteChip renders a cited note path as a clickable accent chip; the click
// opens the note via the consumer's onOpenNote handler.
function NoteChip({ path, onOpenNote }: { path: string; onOpenNote: (path: string) => void }) {
    return (
        <button
            type="button"
            title={`Open ${path}`}
            onClick={() => onOpenNote(path)}
            className="inline-block cursor-pointer align-baseline rounded font-mono text-[0.85em] text-accent underline decoration-transparent underline-offset-2 transition-colors hover:text-accent-hover hover:underline"
        >
            {path}
        </button>
    )
}

// Markdown renders GFM markdown with Shiki-highlighted code blocks inside the
// shared prose wrapper. `trailing` renders after the content (e.g. the
// streaming caret in chat); `className` is appended to the wrapper. When
// `onOpenNote` is provided, inline code matching a wiki note path renders as a
// clickable chip; everything else keeps the default code rendering.
export function Markdown({
    children,
    trailing,
    className = '',
    onOpenNote
}: {
    children: string
    trailing?: ReactNode
    className?: string
    onOpenNote?: (path: string) => void
}) {
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
            const inline = typeof children === 'string' ? children : ''
            if (onOpenNote && NOTE_PATH.test(inline)) return <NoteChip path={inline} onOpenNote={onOpenNote} />
            return <code className={className}>{children}</code>
        }
    }

    return (
        <div className={`${PROSE} ${className}`}>
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
                {children}
            </ReactMarkdown>
            {trailing}
        </div>
    )
}
