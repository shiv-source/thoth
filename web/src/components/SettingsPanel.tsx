import { useEffect, useState, type FormEvent } from 'react'
import { api, type Settings } from '../api/client'

const blank: Settings = { wiki_path: '', host: '127.0.0.1', port: 8333, claude_bin: '', permission_mode: '', model: '' }

export function SettingsPanel() {
  const [form, setForm] = useState<Settings>(blank)
  const [status, setStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')

  useEffect(() => {
    api.settings().then(setForm).catch(() => setStatus('error'))
  }, [])

  const set = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setStatus('saving')
    try {
      const saved = await api.saveSettings(form)
      setForm(saved)
      setStatus('saved')
      setTimeout(() => setStatus('idle'), 2000)
    } catch {
      setStatus('error')
    }
  }

  const field = 'w-full rounded-lg border border-paper-300 bg-white px-3 py-2 text-sm outline-none focus:border-accent-500 dark:border-night-700 dark:bg-night-900'
  const label = 'mb-1 block text-xs font-medium uppercase tracking-wide text-ink-500'

  return (
    <form onSubmit={(e) => { void submit(e) }} className="space-y-4">
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
      <button type="submit" disabled={status === 'saving'}
        className="w-full rounded-lg bg-accent-600 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-accent-700 disabled:opacity-40">
        {status === 'saving' ? 'Saving…' : 'Save settings'}
      </button>
      {status === 'saved' && <p className="text-xs text-green-700 dark:text-green-400">Saved. A wiki path change rebuilds the index.</p>}
      {status === 'error' && <p className="text-xs text-red-600 dark:text-red-400">Could not save settings.</p>}
    </form>
  )
}
