import { useState, type FormEvent } from 'react'

export function Composer({ onSend, onCancel, streaming }: {
  onSend: (text: string) => void
  onCancel: () => void
  streaming: boolean
}) {
  const [text, setText] = useState('')

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const t = text.trim()
    if (!t || streaming) return
    setText('')
    onSend(t)
  }

  return (
    <form onSubmit={submit} className="flex items-end gap-2 border-t border-paper-200 bg-paper-50 p-4 dark:border-night-800 dark:bg-night-950">
      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(e) } }}
        rows={2}
        placeholder="Ask your wiki anything — or tell Thoth to save something…"
        className="flex-1 resize-none rounded-xl border border-paper-300 bg-white px-4 py-3 text-sm outline-none placeholder:text-ink-400 focus:border-accent-500 focus:ring-2 focus:ring-accent-100 dark:border-night-700 dark:bg-night-900 dark:placeholder:text-ink-500 dark:focus:ring-accent-700/20"
      />
      {streaming ? (
        <button type="button" onClick={onCancel}
          className="rounded-xl bg-accent-600 px-4 py-3 text-sm font-medium text-white transition hover:bg-accent-700">
          Stop
        </button>
      ) : (
        <button type="submit" disabled={!text.trim()}
          className="rounded-xl bg-accent-600 px-4 py-3 text-sm font-medium text-white transition hover:bg-accent-700 disabled:cursor-not-allowed disabled:opacity-40">
          Send
        </button>
      )}
    </form>
  )
}
