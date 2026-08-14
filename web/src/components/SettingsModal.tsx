import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { api, type DoctorCheck, type GitHubIdentity, type Settings } from '../api/client'
import { useToast } from './Toast'

const blank: Settings = {
  wiki_path: '', host: '127.0.0.1', port: 8333, claude_bin: '', permission_mode: '', model: '', repo_url: '',
}

const emptyGitHub: GitHubIdentity = {
  username: '', display_name: '', email: '', avatar_url: '', profile_url: '', scopes: '', account_created_at: '', account_updated_at: '',
}

type Tab = 'general' | 'doctor' | 'git'

const tabs: { id: Tab; label: string }[] = [
  { id: 'general', label: 'General' },
  { id: 'doctor', label: 'Doctor' },
  { id: 'git', label: 'Git remote' },
]

const field = 'w-full rounded-lg border border-line bg-app px-3 py-2 text-sm text-ink outline-none placeholder:text-subtle focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500'
const label = 'mb-1 block text-xs font-medium uppercase tracking-wide text-subtle'

export function SettingsModal({ onClose }: { onClose: () => void }) {
  const [tab, setTab] = useState<Tab>('general')
  const [form, setForm] = useState<Settings>(blank)
  const [github, setGitHub] = useState<GitHubIdentity>(emptyGitHub)
  const [status, setStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const { toast } = useToast()

  useEffect(() => {
    Promise.all([api.settings(), api.githubAuth()])
      .then(([s, g]) => { setForm(s); setGitHub(g) })
      .catch(() => setStatus('error'))
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  useEffect(() => () => { if (savedTimer.current) clearTimeout(savedTimer.current) }, [])

  const set = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  const save = async () => {
    setStatus('saving')
    try {
      const saved = await api.saveSettings(form)
      setForm(saved)
      setStatus('saved')
      toast('Settings saved', 'success')
      savedTimer.current = setTimeout(() => setStatus('idle'), 2000)
    } catch {
      setStatus('error')
      toast('Could not save settings', 'error')
    }
  }

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    void save()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm"
      onClick={onClose}>
      <div role="dialog" aria-modal="true" aria-label="Settings"
        onClick={(e) => e.stopPropagation()}
        className="flex h-[36rem] max-h-full w-[48rem] max-w-full animate-[pop-in_150ms_ease-out] flex-col rounded-xl border border-line bg-surface shadow-lg">
        <header className="shrink-0 border-b border-line px-5 pt-4">
          <div className="flex items-center justify-between">
            <h2 className="font-display text-lg font-semibold text-heading">Settings</h2>
            <button onClick={onClose} aria-label="Close settings"
              className="rounded-lg p-1.5 text-subtle transition hover:bg-raised hover:text-ink">✕</button>
          </div>
          <nav role="tablist" aria-label="Settings sections" className="mt-1 flex gap-1">
            {tabs.map((t) => (
              <button key={t.id} role="tab" aria-selected={tab === t.id} onClick={() => setTab(t.id)}
                className={`-mb-px border-b-2 px-3 py-2 text-sm transition ${tab === t.id ? 'border-accent font-medium text-ink' : 'border-transparent text-subtle hover:text-ink'}`}>
                {t.label}
              </button>
            ))}
          </nav>
        </header>

        {tab === 'general' && (
          <form onSubmit={handleSubmit} className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
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
        )}
        {tab === 'doctor' && <DoctorTab />}
        {tab === 'git' && <GitTab form={form} set={set} save={save} github={github} setGitHub={setGitHub} />}
      </div>
    </div>
  )
}

// DoctorTab runs the shared installation checks (GET /api/doctor) on open and
// on demand, rendering each as a row with a green ✓ or red ✗ badge.
function DoctorTab() {
  const [checks, setChecks] = useState<DoctorCheck[] | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async () => {
    setRunning(true)
    setError(null)
    try {
      const res = await api.doctor()
      setChecks(res.checks)
    } catch {
      setError('Could not run checks')
    } finally {
      setRunning(false)
    }
  }, [])

  useEffect(() => {
    void run()
  }, [run])

  return (
    <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-subtle">Installation health, using the same checks as <code className="font-mono text-xs">thoth doctor</code>.</p>
        <button onClick={() => void run()} disabled={running}
          className="flex shrink-0 items-center gap-2 rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised disabled:cursor-not-allowed disabled:opacity-60">
          {running && <span aria-hidden="true" className="h-3 w-3 animate-spin rounded-full border-2 border-line border-t-accent" />}
          {running ? 'Running…' : 'Run checks'}
        </button>
      </div>
      {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
      {checks && (
        <ul className="space-y-2">
          {checks.map((c) => (
            <li key={c.name} className="flex items-start gap-2.5 rounded-lg border border-line bg-app px-3 py-2">
              <span aria-hidden="true" className={`mt-px ${c.ok ? 'text-emerald-500' : 'text-red-500'}`}>{c.ok ? '✓' : '✗'}</span>
              <div className="min-w-0">
                <p className="text-sm font-medium text-ink">{c.name}</p>
                <p className="text-xs text-subtle">{c.message}</p>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

// GitTab connects a GitHub account (the token is stored server-side), keeps
// the sync repo URL in the settings form (Save persists it to the DB), and
// "Initialize & Push" runs the server-side setup against the current wiki
// path. The URL input only appears once connected: the server stores it in
// the github_auth row.
function GitTab({ form, set, save, github, setGitHub }: {
  form: Settings
  set: <K extends keyof Settings>(key: K, value: Settings[K]) => void
  save: () => Promise<void>
  github: GitHubIdentity
  setGitHub: (g: GitHubIdentity) => void
}) {
  const { toast } = useToast()
  const [pushing, setPushing] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [token, setToken] = useState('')
  const [gitError, setGitError] = useState<string | null>(null)

  const connected = github.username !== ''

  const connect = async () => {
    if (!token) {
      setGitError('Enter a personal access token.')
      return
    }
    setConnecting(true)
    setGitError(null)
    try {
      setGitHub(await api.connectGitHub(token))
      setToken('')
      toast('GitHub connected', 'success')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Could not connect GitHub'
      setGitError(msg)
      toast(msg, 'error')
    } finally {
      setConnecting(false)
    }
  }

  const disconnect = async () => {
    try {
      await api.disconnectGitHub()
    } catch {
      toast('Could not disconnect GitHub', 'error')
      return
    }
    setGitHub(emptyGitHub)
    set('repo_url', '')
    toast('GitHub disconnected', 'success')
  }

  const push = async () => {
    if (!form.repo_url) {
      setGitError('Enter a remote URL first.')
      return
    }
    setPushing(true)
    setGitError(null)
    try {
      const res = await api.gitSetup(form.repo_url)
      if (res.ok) {
        toast('Wiki pushed to remote', 'success')
      } else {
        const msg = res.error ?? 'Could not push the wiki'
        setGitError(msg)
        toast(msg, 'error')
      }
    } catch {
      setGitError('Could not reach the server')
      toast('Could not push the wiki', 'error')
    } finally {
      setPushing(false)
    }
  }

  if (!connected) {
    return (
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
        <div>
          <label className={label}>Personal access token</label>
          <input className={field} type="password" placeholder="ghp_…"
            value={token} onChange={(e) => setToken(e.target.value)} />
        </div>
        <p className="text-xs text-subtle">
          Connect your GitHub account to store the sync repo URL and
          credentials. The token needs the <code>user:email</code> scope and
          is stored locally in thoth.db — it is never sent anywhere except
          api.github.com.
        </p>
        {gitError && <p className="text-sm text-red-600 dark:text-red-400">{gitError}</p>}
        <div className="flex items-center justify-end border-t border-line pt-4">
          <button onClick={() => void connect()} disabled={connecting}
            className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-60">
            {connecting && <span aria-hidden="true" className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent-ink/40 border-t-accent-ink" />}
            {connecting ? 'Connecting…' : 'Connect'}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
      <div className="flex items-center gap-3 rounded-lg border border-line bg-app px-3 py-2.5">
        {github.avatar_url !== '' && (
          <img src={github.avatar_url} alt="" aria-hidden="true" className="h-9 w-9 rounded-full" />
        )}
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-ink">
            {github.profile_url !== ''
              ? <a href={github.profile_url} target="_blank" rel="noreferrer" className="hover:underline">{github.display_name || github.username}</a>
              : (github.display_name || github.username)}
          </p>
          <p className="truncate text-xs text-subtle">{github.email || github.username}</p>
          {(github.account_created_at !== '' || github.account_updated_at !== '') && (
            <p className="mt-0.5 truncate text-xs text-subtle">
              Member since {github.account_created_at.slice(0, 10)}
              {github.account_updated_at !== '' && ` · Updated ${github.account_updated_at.slice(0, 10)}`}
            </p>
          )}
        </div>
        <button onClick={() => void disconnect()}
          className="shrink-0 rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised">
          Disconnect
        </button>
      </div>
      <div>
        <label className={label}>Git remote URL</label>
        <input className={field} placeholder="https://github.com/you/wiki.git"
          value={form.repo_url} onChange={(e) => set('repo_url', e.target.value)} />
      </div>
      <p className="text-xs text-subtle">
        Stores your wiki in a remote git repository. Thoth initializes the
        repo if needed, commits the current tree, and pushes the branch.
      </p>
      {gitError && <p className="text-sm text-red-600 dark:text-red-400">{gitError}</p>}
      <div className="flex items-center justify-end gap-3 border-t border-line pt-4">
        <button onClick={() => void save()}
          className="rounded-lg border border-line px-4 py-2 text-sm font-medium text-ink transition hover:bg-raised">
          Save
        </button>
        <button onClick={() => void push()} disabled={pushing}
          className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-60">
          {pushing && <span aria-hidden="true" className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent-ink/40 border-t-accent-ink" />}
          {pushing ? 'Pushing…' : 'Initialize & Push'}
        </button>
      </div>
    </div>
  )
}
