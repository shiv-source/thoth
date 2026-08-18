import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, App, AutoComplete, Avatar, Button, Card, Form, Input, List, Select, Switch, Tabs } from 'antd'
import {
    BranchesOutlined,
    GlobalOutlined,
    LockOutlined,
    ReloadOutlined,
    SettingOutlined as SettingsIcon,
    MedicineBoxOutlined
} from '@ant-design/icons'
import type { GitHubRepo, Settings } from '../api/client'
import {
    connectGit,
    disconnectGit,
    fetchGitAuth,
    fetchGitRepos,
    fetchModels,
    fetchSettings,
    pushWiki,
    runDoctor,
    saveSettings,
    selectDoctorChecks,
    selectDoctorError,
    selectDoctorRunning,
    selectGitAuth,
    selectGitConnecting,
    selectGitError,
    selectGitPushing,
    selectGitRepos,
    selectModels,
    selectSettings
} from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { AppHeader } from './AppHeader'
import { navigateSegment, useViewRoute } from '../hooks/useView'

type Tab = 'general' | 'doctor' | 'git'

const tabs: { id: Tab; label: string; icon: typeof SettingsIcon }[] = [
    { id: 'general', label: 'General', icon: SettingsIcon },
    { id: 'git', label: 'Git remote', icon: BranchesOutlined },
    { id: 'doctor', label: 'Doctor', icon: MedicineBoxOutlined }
]

export function SettingsView() {
    const { segment } = useViewRoute()
    // The active tab rides the URL (#/settings/<tab>) so it survives a
    // reload; an unknown or missing segment falls back to General — and the
    // default is written into the URL so the route is always explicit.
    const [tab, setTab] = useState<Tab>(() => (tabs.some((t) => t.id === segment) ? (segment as Tab) : 'general'))

    useEffect(() => {
        if (segment !== null && tabs.some((t) => t.id === segment)) {
            setTab(segment as Tab)
        } else {
            navigateSegment('settings', 'general')
        }
    }, [segment])

    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)
    const { message } = App.useApp()
    const [form] = Form.useForm<Settings>()
    // The save-feedback banner (Saved ✓ / error) is transient UI, not
    // shared state — it lives here; the store carries loading + data.
    const [status, setStatus] = useState<'idle' | 'saved' | 'error'>('idle')
    const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

    useEffect(() => {
        void dispatch(fetchSettings())
        void dispatch(fetchGitAuth())
        void dispatch(fetchModels())
    }, [dispatch])

    // Seed the form when the store's settings arrive or are saved back;
    // setFieldsValue only touches the named fields, so mid-edit typing is
    // safe.
    useEffect(() => {
        if (settings.data) form.setFieldsValue(settings.data)
    }, [settings.data, form])

    useEffect(
        () => () => {
            if (savedTimer.current) clearTimeout(savedTimer.current)
        },
        []
    )

    const save = async (values: Settings) => {
        try {
            await dispatch(saveSettings(values)).unwrap()
            setStatus('saved')
            void message.success('Settings saved')
            savedTimer.current = setTimeout(() => setStatus('idle'), 2000)
        } catch {
            setStatus('error')
            void message.error('Could not save settings')
        }
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <AppHeader title="Settings" />
            <div className="flex min-h-0 w-full flex-1 flex-col px-6 py-4">
                <Form form={form} layout="vertical" onFinish={(values) => void save(values)} className="min-h-0">
                    <Tabs
                        tabPosition="left"
                        activeKey={tab}
                        onChange={(key) => {
                            setTab(key as Tab)
                            navigateSegment('settings', key)
                        }}
                        items={tabs.map((t) => ({
                            key: t.id,
                            label: (
                                <span className="flex items-center gap-2">
                                    <t.icon aria-hidden="true" />
                                    {t.label}
                                </span>
                            ),
                            children:
                                t.id === 'general' ? (
                                    <GeneralTab status={status} />
                                ) : t.id === 'doctor' ? (
                                    <DoctorTab />
                                ) : (
                                    <GitTab />
                                )
                        }))}
                    />
                </Form>
            </div>
        </div>
    )
}

// GeneralTab is the wiki path + model picker, with the save button and the
// saved/error feedback under them.
function GeneralTab({ status }: { status: 'idle' | 'saved' | 'error' }) {
    const settings = useAppSelector(selectSettings)
    const models = useAppSelector(selectModels)

    // Group the flat model list by provider (stable first-seen order) so the
    // select renders one group per provider; an empty list falls back to the
    // single CLI-default option.
    const modelGroups = useMemo(() => {
        const list = models.length ? models : [{ value: '', label: 'CLI default', provider: 'Claude Code' }]
        const order: string[] = []
        const byProvider = new Map<string, typeof list>()
        for (const m of list) {
            if (!byProvider.has(m.provider)) {
                byProvider.set(m.provider, [])
                order.push(m.provider)
            }
            byProvider.get(m.provider)!.push(m)
        }
        return order.map((provider) => ({ label: provider, options: byProvider.get(provider)! }))
    }, [models])

    return (
        <div className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
                <Card title="Wiki" size="small">
                    <Form.Item
                        label="Wiki path"
                        name="wiki_path"
                        extra="Where your notes live on disk. Defaults to ~/.thoth/wiki."
                    >
                        <Input placeholder="~/.thoth/wiki" />
                    </Form.Item>
                </Card>
                <Card title="AI model" size="small">
                    <Form.Item label="Model" name="model" extra="Applied to all chats after the app restarts.">
                        <Select virtual={false} options={modelGroups} />
                    </Form.Item>
                </Card>
            </div>
            <div className="flex items-center justify-between">
                <div className="min-w-0 pr-3">
                    {status === 'saved' && <Alert type="success" showIcon message="Saved ✓" />}
                    {(status === 'error' || settings.error !== null) && (
                        <Alert type="error" showIcon message="Could not save settings." />
                    )}
                </div>
                <Button type="primary" htmlType="submit" loading={settings.saving}>
                    Save
                </Button>
            </div>
        </div>
    )
}

// DoctorTab runs the shared installation checks (GET /api/doctor) on open
// and on demand, rendering each as a list row.
function DoctorTab() {
    const dispatch = useAppDispatch()
    const checks = useAppSelector(selectDoctorChecks)
    const running = useAppSelector(selectDoctorRunning)
    const error = useAppSelector(selectDoctorError)

    useEffect(() => {
        void dispatch(runDoctor())
    }, [dispatch])

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between gap-3">
                <p className="text-sm text-subtle">
                    Installation health, using the same checks as{' '}
                    <code className="font-mono text-xs">thoth doctor</code>.
                </p>
                <Button
                    icon={<ReloadOutlined aria-hidden="true" />}
                    loading={running}
                    onClick={() => void dispatch(runDoctor())}
                >
                    Run checks
                </Button>
            </div>
            {error && <Alert type="error" showIcon message={error} />}
            {checks && (
                <Card title="Checks" size="small">
                    <List
                        size="small"
                        dataSource={checks}
                        renderItem={(c) => (
                            <List.Item>
                                <List.Item.Meta
                                    avatar={
                                        <span aria-hidden="true" className={c.ok ? 'text-emerald-500' : 'text-red-500'}>
                                            {c.ok ? '✓' : '✗'}
                                        </span>
                                    }
                                    title={c.name}
                                    description={c.message}
                                />
                            </List.Item>
                        )}
                    />
                </Card>
            )}
        </div>
    )
}

// GitTab connects a GitHub account (the token is stored server-side), keeps
// the sync repo URL in the settings form (Save persists it to the DB), and
// "Initialize & Push" runs the server-side setup against the current wiki
// path. The URL input only appears once connected: the server stores it in
// the github_auth row.
function GitTab() {
    const dispatch = useAppDispatch()
    const { message } = App.useApp()
    const form = Form.useFormInstance<Settings>()
    const auth = useAppSelector(selectGitAuth)
    const repos = useAppSelector(selectGitRepos)
    const connecting = useAppSelector(selectGitConnecting)
    const pushing = useAppSelector(selectGitPushing)
    const gitError = useAppSelector(selectGitError)
    const [token, setToken] = useState('')
    // Only a repo picked from the suggestions is classified: a hand-typed
    // URL gets no visibility warning (it cannot be verified here).
    const [selectedRepo, setSelectedRepo] = useState<GitHubRepo | null>(null)

    const connected = auth !== null && auth.username !== ''

    // Suggestions for the repo URL come from the connected account; a failed
    // load just leaves the list empty — typing a URL always works.
    useEffect(() => {
        if (connected) void dispatch(fetchGitRepos())
    }, [connected, dispatch])

    const repoOptions = useMemo(
        () =>
            (repos ?? []).map((r) => ({
                value: r.clone_url,
                repo: r,
                label: (
                    <span className="flex items-start gap-2">
                        {r.private ? (
                            <LockOutlined aria-hidden="true" className="mt-0.5 shrink-0 text-subtle" />
                        ) : (
                            <GlobalOutlined aria-hidden="true" className="mt-0.5 shrink-0 text-subtle" />
                        )}
                        <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm">{r.full_name}</span>
                            {r.description !== '' && (
                                <span className="block truncate text-xs text-subtle">{r.description}</span>
                            )}
                        </span>
                    </span>
                )
            })),
        [repos]
    )

    const publicSelected = selectedRepo !== null && !selectedRepo.private

    const connect = async () => {
        if (!token) {
            void message.error('Enter a personal access token.')
            return
        }
        try {
            await dispatch(connectGit(token)).unwrap()
            setToken('')
            void message.success('GitHub connected')
        } catch (e) {
            void message.error(e instanceof Error ? e.message : 'Could not connect GitHub')
        }
    }

    const disconnect = async () => {
        try {
            await dispatch(disconnectGit()).unwrap()
            form.setFieldsValue({ repo_url: '' })
            void message.success('GitHub disconnected')
        } catch {
            void message.error('Could not disconnect GitHub')
        }
    }

    const push = async () => {
        const url = form.getFieldValue('repo_url') as string
        if (!url) {
            void message.error('Enter a remote URL first.')
            return
        }
        try {
            const res = await dispatch(pushWiki(url)).unwrap()
            if (res.ok) {
                void message.success('Wiki pushed to remote')
            } else {
                void message.error(res.error ?? 'Could not push the wiki')
            }
        } catch {
            void message.error('Could not push the wiki')
        }
    }

    if (!connected) {
        return (
            <Card title="GitHub account" size="small" className="max-w-xl">
                <Form.Item
                    label="Personal access token"
                    extra={
                        <>
                            Connect your GitHub account to store the sync repo URL and credentials. The token needs the{' '}
                            <code>user:email</code> scope and is stored locally in thoth.db — it is never sent anywhere
                            except api.github.com.
                        </>
                    }
                >
                    <Input.Password placeholder="ghp_…" value={token} onChange={(e) => setToken(e.target.value)} />
                </Form.Item>
                {gitError && <Alert type="error" showIcon message={gitError} className="mb-4" />}
                <div className="flex justify-end">
                    <Button type="primary" loading={connecting} onClick={() => void connect()}>
                        Connect
                    </Button>
                </div>
            </Card>
        )
    }

    return (
        <div className="max-w-xl space-y-4">
            <Card title="GitHub account" size="small">
                <div className="flex items-center gap-3">
                    {auth.avatar_url !== '' && <Avatar src={auth.avatar_url} size={36} />}
                    <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-ink">
                            {auth.profile_url !== '' ? (
                                <a href={auth.profile_url} target="_blank" rel="noreferrer" className="hover:underline">
                                    {auth.display_name || auth.username}
                                </a>
                            ) : (
                                auth.display_name || auth.username
                            )}
                        </p>
                        <p className="truncate text-xs text-subtle">{auth.email || auth.username}</p>
                        {(auth.account_created_at !== '' || auth.account_updated_at !== '') && (
                            <p className="mt-0.5 truncate text-xs text-subtle">
                                Member since {auth.account_created_at.slice(0, 10)}
                                {auth.account_updated_at !== '' && ` · Updated ${auth.account_updated_at.slice(0, 10)}`}
                            </p>
                        )}
                    </div>
                    <Button onClick={() => void disconnect()}>Disconnect</Button>
                </div>
            </Card>
            <Card title="Remote" size="small">
                <Form.Item label="Git remote URL" name="repo_url">
                    <AutoComplete
                        virtual={false}
                        placeholder="https://github.com/you/wiki.git"
                        options={repoOptions}
                        onSelect={(_value, option) => {
                            const repo = (option as { repo?: GitHubRepo }).repo
                            setSelectedRepo(repo ?? null)
                        }}
                        onChange={(value) => {
                            if (selectedRepo && selectedRepo.clone_url !== value) setSelectedRepo(null)
                        }}
                    />
                </Form.Item>
                {publicSelected && (
                    <Alert
                        type="error"
                        showIcon
                        message="Syncing to a public repository is blocked for your security — use a private repository."
                    />
                )}
            </Card>
            <Card title="Sync" size="small">
                <Form.Item
                    name="sync_enabled"
                    valuePropName="checked"
                    label="Auto-sync the wiki to the remote"
                    extra="Stores your wiki in a remote git repository. Thoth initializes the repo if needed, commits the current tree, and pushes the branch."
                >
                    <Switch />
                </Form.Item>
            </Card>
            {gitError && <Alert type="error" showIcon message={gitError} />}
            <div className="flex justify-end gap-3">
                <Button htmlType="submit">Save</Button>
                <Button type="primary" loading={pushing} disabled={publicSelected} onClick={() => void push()}>
                    Initialize & Push
                </Button>
            </div>
        </div>
    )
}
