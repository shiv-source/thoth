import { useEffect, useRef, useState, type FormEvent } from 'react'
import { api, type Settings } from '../api/client'

const blank: Settings = { wiki_path: '', host: '127.0.0.1', port: 8333, claude_bin: '', permission_mode: '', model: '' }

export function SettingsModal({ onClose }: { onClose: () => void }) {
  const [form, setForm] = useState<Settings>(blank)
  const [status, setStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    api.settings().then(setForm).catch(() => setStatus('error'))
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  useEffect(() => () => { if (savedTimer.current) clearTimeout(savedTimer.current) }, [])

  const set = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setStatus('saving')
    try {
      const saved = await api.saveSettings(form)
      setForm(saved)
      setStatus('saved')
      savedTimer.current = setTimeout(() => setStatus('idle'), 2000)
    } catch {
      setStatus('error')
    }
  }

  const field = 'w-full rounded-lg border border-line bg-app px-3 py-2 text-sm text-ink outline-none placeholder:text-subtle focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500'
  const label = 'mb-1 block text-xs font-medium uppercase tracking-wide text-subtle'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm"
      onClick={onClose}>
      <div role="dialog" aria-modal="true" aria-label="Settings"
        onClick={(e) => e.stopPropagation()}
        className="flex max-h-full w-full max-w-[28rem] animate-[pop-in_150ms_ease-out] flex-col rounded-xl border border-line bg-surface shadow-lg">
        <header className="flex shrink-0 items-center justify-between border-b border-line px-5 py-4">
          <h2 className="font-display text-lg font-semibold text-heading">Settings</h2>
          <button onClick={onClose} aria-label="Close settings"
            className="rounded-lg p-1.5 text-subtle transition hover:bg-raised hover:text-ink">✕</button>
        </header>
        <form onSubmit={(e) => { void submit(e) }} className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
          <div>
            <label className={label}>Wiki path</label>
            <input className={field} value={form.wiki_path} onChange={(e) => set('wiki_path', e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className={label}>Host</label>
              <input className={field} value={form.host} onChange={(e) => set('host', e.target.value)} />
            </div>
            <div>
              <label className={label}>Port</label>
              <input className={field} type="number" value={form.port} onChange={(e) => set('port', Number(e.target.value))} />
            </div>
          </div>
          <div>
            <label className={label}>Claude binary</label>
            <input className={field} placeholder="claude (from PATH)" value={form.claude_bin} onChange={(e) => set('claude_bin', e.target.value)} />
          </div>
          <div>
            <label className={label}>Permission mode</label>
            <input className={field} placeholder="safe default" value={form.permission_mode} onChange={(e) => set('permission_mode', e.target.value)} />
          </div>
          <div>
            <label className={label}>Model</label>
            <input className={field} placeholder="CLI default" value={form.model} onChange={(e) => set('model', e.target.value)} />
          </div>
          <div className="flex items-center justify-between border-t border-line pt-4">
            <div className="min-w-0 pr-3 text-xs">
              {status === 'saved' && <p className="text-emerald-600 dark:text-emerald-400">Saved ✓</p>}
              {status === 'error' && <p className="text-red-600 dark:text-red-400">Could not save settings.</p>}
            </div>
            <button type="submit" disabled={status === 'saving'}
              className="shrink-0 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-40">
              {status === 'saving' ? 'Saving…' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
