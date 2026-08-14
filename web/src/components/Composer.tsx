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
    if (!t) return
    // Sending while a turn streams is allowed: the server cancels the
    // in-flight turn and starts the new one (supersede).
    setText('')
    onSend(t)
  }

  return (
    <form onSubmit={submit} className="flex shrink-0 items-end gap-2 border-t border-line bg-app px-4 py-4">
      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(e) } }}
        rows={2}
        placeholder="Ask your wiki anything — or tell Thoth to save something…"
        className="flex-1 resize-none rounded-xl border border-line bg-surface px-4 py-3 text-sm text-ink outline-none placeholder:text-subtle focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500"
      />
      {streaming ? (
        <button type="button" onClick={onCancel}
          className="shrink-0 rounded-xl border border-line px-4 py-3 text-sm font-medium text-ink transition hover:bg-raised">
          Stop
        </button>
      ) : (
        <button type="submit" disabled={!text.trim()}
          className="shrink-0 rounded-xl bg-accent px-4 py-3 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-40">
          Send
        </button>
      )}
    </form>
  )
}
