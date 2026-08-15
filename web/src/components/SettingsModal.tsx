import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { api, type DoctorCheck, type GitHubIdentity, type GitHubRepo, type Settings } from '../api/client'
import { fetchSettings, saveSettings, selectSettings } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { useToast } from './Toast'

const blank: Settings = {
    wiki_path: '',
    repo_url: '',
    sync_enabled: false
}

const emptyGitHub: GitHubIdentity = {
    username: '',
    display_name: '',
    email: '',
    avatar_url: '',
    profile_url: '',
    scopes: '',
    account_created_at: '',
    account_updated_at: ''
}

type Tab = 'general' | 'doctor' | 'git'

const tabs: { id: Tab; label: string }[] = [
    { id: 'general', label: 'General' },
    { id: 'git', label: 'Git remote' },
    { id: 'doctor', label: 'Doctor' }
]

const field =
    'w-full rounded-lg border border-line bg-app px-3 py-2 text-sm text-ink outline-none placeholder:text-subtle focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500'
const label = 'mb-1 block text-xs font-medium uppercase tracking-wide text-subtle'

export function SettingsModal({ onClose }: { onClose: () => void }) {
    const [tab, setTab] = useState<Tab>('general')
    const [form, setForm] = useState<Settings>(blank)
    const [github, setGitHub] = useState<GitHubIdentity>(emptyGitHub)
    // The 'saving' state lives in the settings slice — the button reads it.
    const [status, setStatus] = useState<'idle' | 'saved' | 'error'>('idle')
    const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
    const { toast } = useToast()
    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)

    useEffect(() => {
        void dispatch(fetchSettings())
        api.githubAuth()
            .then(setGitHub)
            .catch(() => setStatus('error'))
    }, [dispatch])

    // Seed the form when the store's settings arrive or are saved back; the
    // guard keeps mid-edit typing safe (this only runs when data changes).
    useEffect(() => {
        if (settings.data) setForm(settings.data)
    }, [settings.data])

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose()
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    }, [onClose])

    useEffect(
        () => () => {
            if (savedTimer.current) clearTimeout(savedTimer.current)
        },
        []
    )

    const set = <K extends keyof Settings>(key: K, value: Settings[K]) => setForm((f) => ({ ...f, [key]: value }))

    const save = async () => {
        try {
            const saved = await dispatch(saveSettings(form)).unwrap()
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
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4 backdrop-blur-sm"
            onClick={onClose}
        >
            <div
                role="dialog"
                aria-modal="true"
                aria-label="Settings"
                onClick={(e) => e.stopPropagation()}
                className="flex h-[36rem] max-h-full w-[48rem] max-w-full animate-[pop-in_150ms_ease-out] flex-col rounded-xl border border-line bg-surface shadow-lg"
            >
                <header className="shrink-0 border-b border-line px-5 pt-4">
                    <div className="flex items-center justify-between">
                        <h2 className="font-display text-lg font-semibold text-heading">Settings</h2>
                        <button
                            onClick={onClose}
                            aria-label="Close settings"
                            className="rounded-lg p-1.5 text-subtle transition hover:bg-raised hover:text-ink"
                        >
                            ✕
                        </button>
                    </div>
                    <nav role="tablist" aria-label="Settings sections" className="mt-1 flex gap-1">
                        {tabs.map((t) => (
                            <button
                                key={t.id}
                                role="tab"
                                aria-selected={tab === t.id}
                                onClick={() => setTab(t.id)}
                                className={`-mb-px border-b-2 px-3 py-2 text-sm transition ${tab === t.id ? 'border-accent font-medium text-ink' : 'border-transparent text-subtle hover:text-ink'}`}
                            >
                                {t.label}
                            </button>
                        ))}
                    </nav>
                </header>

                {tab === 'general' && (
                    <form onSubmit={handleSubmit} className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
                        <div>
                            <label className={label}>Wiki path</label>
                            <input
                                className={field}
                                placeholder="~/.thoth/wiki"
                                value={form.wiki_path}
                                onChange={(e) => set('wiki_path', e.target.value)}
                            />
                        </div>
                        <div className="flex items-center justify-between border-t border-line pt-4">
                            <div className="min-w-0 pr-3 text-xs">
                                {status === 'saved' && (
                                    <p className="text-emerald-600 dark:text-emerald-400">Saved ✓</p>
                                )}
                                {(status === 'error' || settings.error !== null) && (
                                    <p className="text-red-600 dark:text-red-400">Could not save settings.</p>
                                )}
                            </div>
                            <button
                                type="submit"
                                disabled={settings.saving}
                                className="shrink-0 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-40"
                            >
                                {settings.saving ? 'Saving…' : 'Save'}
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
                <p className="text-sm text-subtle">
                    Installation health, using the same checks as{' '}
                    <code className="font-mono text-xs">thoth doctor</code>.
                </p>
                <button
                    onClick={() => void run()}
                    disabled={running}
                    className="flex shrink-0 items-center gap-2 rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised disabled:cursor-not-allowed disabled:opacity-60"
                >
                    {running && (
                        <span
                            aria-hidden="true"
                            className="h-3 w-3 animate-spin rounded-full border-2 border-line border-t-accent"
                        />
                    )}
                    {running ? 'Running…' : 'Run checks'}
                </button>
            </div>
            {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
            {checks && (
                <ul className="space-y-2">
                    {checks.map((c) => (
                        <li
                            key={c.name}
                            className="flex items-start gap-2.5 rounded-lg border border-line bg-app px-3 py-2"
                        >
                            <span aria-hidden="true" className={`mt-px ${c.ok ? 'text-emerald-500' : 'text-red-500'}`}>
                                {c.ok ? '✓' : '✗'}
                            </span>
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
function GitTab({
    form,
    set,
    save,
    github,
    setGitHub
}: {
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
    const [repos, setRepos] = useState<GitHubRepo[]>([])
    const [gitError, setGitError] = useState<string | null>(null)
    const [repoOpen, setRepoOpen] = useState(false)
    const [selectedRepo, setSelectedRepo] = useState<GitHubRepo | null>(null)

    const connected = github.username !== ''

    // Repos matching the typed URL prefix, shown in a dropdown the same width
    // as the input.
    const query = form.repo_url.trim().toLowerCase()
    const filteredRepos = repos.filter(
        (r) => r.full_name.toLowerCase().includes(query) || r.clone_url.toLowerCase().includes(query)
    )
    // Only a repo picked from the suggestions is classified: a hand-typed URL
    // gets no visibility warning (it cannot be verified here).
    const publicSelected = selectedRepo !== null && !selectedRepo.private

    const pickRepo = (r: GitHubRepo) => {
        set('repo_url', r.clone_url)
        setSelectedRepo(r)
        setRepoOpen(false)
    }

    // Suggestions for the repo URL come from the connected account; a failed
    // load just leaves the list empty — typing a URL always works.
    useEffect(() => {
        if (!connected) return
        api.githubRepos()
            .then((r) => setRepos(r.repos))
            .catch(() => setRepos([]))
    }, [connected])

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
                    <input
                        className={field}
                        type="password"
                        placeholder="ghp_…"
                        value={token}
                        onChange={(e) => setToken(e.target.value)}
                    />
                </div>
                <p className="text-xs text-subtle">
                    Connect your GitHub account to store the sync repo URL and credentials. The token needs the{' '}
                    <code>user:email</code> scope and is stored locally in thoth.db — it is never sent anywhere except
                    api.github.com.
                </p>
                {gitError && <p className="text-sm text-red-600 dark:text-red-400">{gitError}</p>}
                <div className="flex items-center justify-end border-t border-line pt-4">
                    <button
                        onClick={() => void connect()}
                        disabled={connecting}
                        className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-60"
                    >
                        {connecting && (
                            <span
                                aria-hidden="true"
                                className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent-ink/40 border-t-accent-ink"
                            />
                        )}
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
                        {github.profile_url !== '' ? (
                            <a href={github.profile_url} target="_blank" rel="noreferrer" className="hover:underline">
                                {github.display_name || github.username}
                            </a>
                        ) : (
                            github.display_name || github.username
                        )}
                    </p>
                    <p className="truncate text-xs text-subtle">{github.email || github.username}</p>
                    {(github.account_created_at !== '' || github.account_updated_at !== '') && (
                        <p className="mt-0.5 truncate text-xs text-subtle">
                            Member since {github.account_created_at.slice(0, 10)}
                            {github.account_updated_at !== '' && ` · Updated ${github.account_updated_at.slice(0, 10)}`}
                        </p>
                    )}
                </div>
                <button
                    onClick={() => void disconnect()}
                    className="shrink-0 rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-raised"
                >
                    Disconnect
                </button>
            </div>
            <div className="relative">
                <label className={label}>Git remote URL</label>
                <input
                    className={field}
                    placeholder="https://github.com/you/wiki.git"
                    value={form.repo_url}
                    onFocus={() => setRepoOpen(true)}
                    onBlur={() => setRepoOpen(false)}
                    onChange={(e) => {
                        set('repo_url', e.target.value)
                        setSelectedRepo((prev) => (prev && prev.clone_url === e.target.value ? prev : null))
                    }}
                />
                {repoOpen && filteredRepos.length > 0 && (
                    <ul className="absolute left-0 right-0 top-full z-10 max-h-56 overflow-y-auto rounded-b-lg border border-line bg-surface shadow-md">
                        {filteredRepos.map((r) => (
                            <li key={r.full_name}>
                                <button
                                    type="button"
                                    onMouseDown={(e) => {
                                        e.preventDefault()
                                        pickRepo(r)
                                    }}
                                    className="flex w-full items-start gap-2 px-3 py-2 text-left transition hover:bg-raised"
                                >
                                    {r.private ? <LockIcon /> : <GlobeIcon />}
                                    <span className="min-w-0 flex-1">
                                        <span className="block truncate text-sm text-ink">{r.full_name}</span>
                                        {r.description !== '' && (
                                            <span className="block truncate text-xs text-subtle">{r.description}</span>
                                        )}
                                    </span>
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
            {publicSelected && (
                <p className="text-sm text-red-600 dark:text-red-400">
                    Syncing to a public repository is blocked for your security — use a private repository.
                </p>
            )}
            <label className="flex items-center gap-2.5 rounded-lg border border-line bg-app px-3 py-2.5 text-sm text-ink">
                <input
                    type="checkbox"
                    checked={form.sync_enabled}
                    onChange={(e) => set('sync_enabled', e.target.checked)}
                    className="h-4 w-4 accent-emerald-500"
                />
                Auto-sync the wiki to the remote
            </label>
            <p className="text-xs text-subtle">
                Stores your wiki in a remote git repository. Thoth initializes the repo if needed, commits the current
                tree, and pushes the branch.
            </p>
            {gitError && <p className="text-sm text-red-600 dark:text-red-400">{gitError}</p>}
            <div className="flex items-center justify-end gap-3 border-t border-line pt-4">
                <button
                    onClick={() => void save()}
                    className="rounded-lg border border-line px-4 py-2 text-sm font-medium text-ink transition hover:bg-raised"
                >
                    Save
                </button>
                <button
                    onClick={() => void push()}
                    disabled={pushing || publicSelected}
                    className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-60"
                >
                    {pushing && (
                        <span
                            aria-hidden="true"
                            className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent-ink/40 border-t-accent-ink"
                        />
                    )}
                    {pushing ? 'Pushing…' : 'Initialize & Push'}
                </button>
            </div>
        </div>
    )
}

function GlobeIcon() {
    return (
        <svg
            viewBox="0 0 20 20"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            aria-hidden="true"
            className="h-4 w-4 shrink-0 text-subtle"
        >
            <circle cx="10" cy="10" r="7.5" />
            <path d="M2.5 10h15M10 2.5c2.5 2 3.8 4.5 3.8 7.5S12.5 15.5 10 17.5c-2.5-2-3.8-4.5-3.8-7.5S7.5 4.5 10 2.5z" />
        </svg>
    )
}

function LockIcon() {
    return (
        <svg
            viewBox="0 0 20 20"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            aria-hidden="true"
            className="h-4 w-4 shrink-0 text-subtle"
        >
            <rect x="4.5" y="8.5" width="11" height="8" rx="1.5" />
            <path d="M7 8.5V6.5a3 3 0 0 1 6 0v2" />
        </svg>
    )
}
