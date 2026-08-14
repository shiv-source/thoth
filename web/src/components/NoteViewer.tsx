import { useEffect, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { api } from '../api/client'

export function NoteViewer({ path, onClose }: { path: string; onClose: () => void }) {
  const [content, setContent] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.note(path).then((n) => setContent(n.content)).catch((e: Error) => setError(e.message))
  }, [path])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink-900/40 p-6 backdrop-blur-sm" onClick={onClose}>
      <div className="h-full max-h-[80vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-paper-200 bg-paper-50 p-6 shadow-2xl dark:border-night-700 dark:bg-night-950"
        onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <span className="truncate font-mono text-xs text-ink-500">{path}</span>
          <button onClick={onClose} className="rounded-lg px-2 py-1 text-sm text-ink-500 hover:bg-paper-200 dark:hover:bg-night-800">✕</button>
        </div>
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
        {content && (
          <div className="prose prose-sm max-w-none prose-headings:font-display prose-headings:text-accent-700 dark:prose-invert dark:prose-headings:text-accent-500">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
          </div>
        )}
      </div>
    </div>
  )
}
