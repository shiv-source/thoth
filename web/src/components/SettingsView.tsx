import { useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import {
    Alert,
    App,
    AutoComplete,
    Avatar,
    Button,
    Card,
    Divider,
    Empty,
    Flex,
    Form,
    Input,
    Modal,
    Popconfirm,
    Progress,
    Select,
    Switch,
    Table,
    Tabs,
    Tag,
    Typography,
    theme
} from 'antd'
import type { TableProps } from 'antd'
import {
    ApiOutlined,
    BranchesOutlined,
    CheckCircleFilled,
    ClockCircleOutlined,
    CloseCircleFilled,
    GithubOutlined,
    GlobalOutlined,
    LockOutlined,
    MedicineBoxOutlined,
    ReloadOutlined,
    SettingOutlined as SettingsIcon,
    SyncOutlined
} from '@ant-design/icons'
import type { DoctorCheck, GitHubIdentity, GitHubRepo, LLMModel, ModelInput, Settings } from '../api/client'
import {
    connectGit,
    createModel,
    deleteModel,
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
    selectHealth,
    selectModelGroups,
    selectModelList,
    selectSettings,
    updateModel
} from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { AppHeader } from './AppHeader'
import { WikiPathInput } from './WikiPathInput'
import { navigateSegment, useViewRoute } from '../hooks/useView'

type Tab = 'general' | 'doctor' | 'git' | 'models'

const tabs: { id: Tab; label: string; icon: typeof SettingsIcon }[] = [
    { id: 'general', label: 'General', icon: SettingsIcon },
    { id: 'models', label: 'LLM Models', icon: ApiOutlined },
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
                        tabPlacement="start"
                        activeKey={tab}
                        // The rail is styled by the Tabs tokens in theme.ts
                        // plus the .settings-tabs pill rules in index.css.
                        classNames={{ root: 'settings-tabs' }}
                        tabBarStyle={{ minWidth: 140 }}
                        // The default left-placement divider (body-holder
                        // border) renders as a stray full-height vertical
                        // line between the tab rail and the content.
                        styles={{ body: { borderLeft: 'none', marginLeft: 0 } }}
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
                                ) : t.id === 'models' ? (
                                    <ModelsTab />
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

// SectionHeading is the icon'd micro-title that opens each card section
// (the DashboardView kicker pattern, inside a Card).
function SectionHeading({ icon: Icon, children }: { icon: typeof SettingsIcon; children: ReactNode }) {
    return (
        <Flex align="center" gap={8} className="mb-1">
            <Icon aria-hidden="true" className="text-subtle" />
            <h3 className="text-xs font-medium uppercase tracking-wide text-subtle">{children}</h3>
        </Flex>
    )
}

// CardTitle is the icon'd Card title shared by every tab's card, so the
// three panes read as one system (General / GitHub / Checks).
function CardTitle({ icon: Icon, children }: { icon: typeof SettingsIcon; children: ReactNode }) {
    return (
        <Flex align="center" gap={8}>
            <Icon aria-hidden="true" className="text-accent" />
            {children}
        </Flex>
    )
}

// GeneralTab is the wiki path + model picker + API key in one card, with the
// save button and the saved/error feedback under them.
function GeneralTab({ status }: { status: 'idle' | 'saved' | 'error' }) {
    const settings = useAppSelector(selectSettings)
    const groups = useAppSelector(selectModelGroups)
    // The server reports the wiki default for the mode it runs in; the
    // fallback covers the render before the first health response lands.
    const defaultWiki = useAppSelector(selectHealth)?.default_wiki_path ?? '~/.thoth/wiki'

    // The server sends models grouped by provider (A→Z); options keep name +
    // tag + value so optionRender can show the tag as secondary text. An
    // empty registry leaves the picker empty — there is no fallback option.
    const modelGroups = useMemo(
        () =>
            groups.map((g) => ({
                label: g.provider,
                options: g.models.map((m) => ({ label: m.name, value: m.value, tag: m.tag }))
            })),
        [groups]
    )

    return (
        <Card size="small" className="max-w-3xl" title={<CardTitle icon={SettingsIcon}>General</CardTitle>}>
            <div className="grid gap-4 md:grid-cols-2">
                <Form.Item
                    label="Wiki path"
                    name="wiki_path"
                    extra={`Where your notes live on disk. Defaults to ${defaultWiki}.`}
                >
                    <WikiPathInput />
                </Form.Item>
                <Form.Item label="Model" name="model" extra="Applied to all chats after the app restarts.">
                    <Select
                        virtual={false}
                        options={modelGroups}
                        notFoundContent="No models — add some in the LLM Models tab"
                        optionRender={(option) => (
                            <Flex align="center" gap={8}>
                                <span className="min-w-0 flex-1 truncate">{option.label}</span>
                                {typeof option.data === 'object' &&
                                    option.data !== null &&
                                    (option.data as { tag?: string }).tag !== '' && (
                                        <span className="shrink-0 text-xs text-subtle">
                                            {(option.data as { tag?: string }).tag}
                                        </span>
                                    )}
                            </Flex>
                        )}
                    />
                </Form.Item>
                <Form.Item
                    label={
                        <Flex align="center" gap={6}>
                            API key
                            {settings.data?.has_api_key ? <Tag color="success">Configured</Tag> : <Tag>Not set</Tag>}
                        </Flex>
                    }
                    name="api_key"
                    extra={
                        settings.data?.has_api_key
                            ? 'A key is saved — leave blank to keep it.'
                            : 'Not set — the server inherits ANTHROPIC_API_KEY from its environment.'
                    }
                >
                    <Input.Password
                        placeholder={settings.data?.has_api_key ? '•••••••• (saved)' : 'sk-ant-…'}
                        autoComplete="off"
                    />
                </Form.Item>
            </div>
            <Divider />
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
        </Card>
    )
}

// tagColor maps the seeded tags to stable antd preset colors so the same
// tag reads the same everywhere in the table; unknown tags fall back to the
// neutral default.
function tagColor(tag: string): string {
    const colors: Record<string, string> = {
        strongest: 'gold',
        flagship: 'blue',
        balanced: 'green',
        fastest: 'cyan',
        fast: 'cyan',
        advanced: 'purple',
        efficient: 'lime',
        reasoning: 'magenta',
        coding: 'geekblue',
        powerful: 'volcano',
        open: 'default'
    }
    return colors[tag] ?? 'default'
}

// ModelsTab is the llm_models CRUD surface: a table of every model plus the
// add/edit modal and per-row delete. Every mutation refetches the registry
// (the server re-groups and re-sorts) and settings (a rename may have moved
// the selected-model setting, a delete may have cleared it).
function ModelsTab() {
    const dispatch = useAppDispatch()
    const models = useAppSelector(selectModelList)
    const groups = useAppSelector(selectModelGroups)
    const { message } = App.useApp()
    const [open, setOpen] = useState(false)
    const [editing, setEditing] = useState<LLMModel | null>(null)
    const [form] = Form.useForm<ModelInput>()

    // The tag dropdown offers every tag already in the registry (sorted);
    // mode="tags" also lets the user type a new one.
    const tagOptions = useMemo(
        () =>
            [...new Set(groups.flatMap((g) => g.models.map((m) => m.tag)).filter((t) => t !== ''))]
                .sort()
                .map((t) => ({ value: t })),
        [groups]
    )

    const openAdd = () => {
        setEditing(null)
        form.resetFields()
        setOpen(true)
    }

    const openEdit = (m: LLMModel) => {
        setEditing(m)
        form.setFieldsValue(m)
        setOpen(true)
    }

    const submit = async () => {
        const values = await form.validateFields()
        try {
            if (editing) {
                await dispatch(updateModel({ id: editing.id, input: values })).unwrap()
            } else {
                await dispatch(createModel(values)).unwrap()
            }
            void dispatch(fetchModels())
            void dispatch(fetchSettings())
            setOpen(false)
            void message.success(editing ? 'Model updated' : 'Model added')
        } catch {
            void message.error('Could not save model')
        }
    }

    const remove = async (m: LLMModel) => {
        try {
            await dispatch(deleteModel(m.id)).unwrap()
            void dispatch(fetchModels())
            void dispatch(fetchSettings())
            void message.success('Model deleted')
        } catch {
            void message.error('Could not delete model')
        }
    }

    const columns: TableProps<LLMModel>['columns'] = [
        { title: 'Name', dataIndex: 'name' },
        {
            title: 'Tag',
            dataIndex: 'tag',
            render: (value: string) =>
                value !== '' ? (
                    <Tag color={tagColor(value)}>{value}</Tag>
                ) : (
                    <Typography.Text type="secondary">—</Typography.Text>
                )
        },
        { title: 'Provider', dataIndex: 'provider' },
        {
            title: 'Value',
            dataIndex: 'value',
            render: (value: string) => <code className="font-mono text-xs">{value}</code>
        },
        {
            title: '',
            key: 'actions',
            width: 150,
            render: (_, m) => (
                <Flex gap={4} justify="flex-end">
                    <Button size="small" onClick={() => openEdit(m)}>
                        Edit
                    </Button>
                    <Popconfirm title="Delete this model?" trigger="click" onConfirm={() => void remove(m)}>
                        <Button size="small" danger>
                            Delete
                        </Button>
                    </Popconfirm>
                </Flex>
            )
        }
    ]

    return (
        <Card
            size="small"
            className="max-w-4xl"
            title={<CardTitle icon={ApiOutlined}>LLM Models</CardTitle>}
            extra={
                <Button type="primary" icon={<ApiOutlined aria-hidden="true" />} onClick={openAdd}>
                    Add model
                </Button>
            }
        >
            <Flex align="center" justify="space-between" className="mb-4">
                <p className="text-sm text-subtle">
                    <Typography.Text strong>{models.length}</Typography.Text> model{models.length === 1 ? '' : 's'}{' '}
                    across <Typography.Text strong>{groups.length}</Typography.Text> provider
                    {groups.length === 1 ? '' : 's'} · the selected model feeds the{' '}
                    <code className="font-mono text-xs">--model</code> CLI flag.
                </p>
            </Flex>
            <Table<LLMModel>
                rowKey="id"
                size="small"
                columns={columns}
                dataSource={models}
                pagination={false}
                locale={{
                    emptyText: (
                        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No models yet">
                            <Button type="primary" onClick={openAdd}>
                                Add your first model
                            </Button>
                        </Empty>
                    )
                }}
            />
            <Modal
                title={editing ? 'Edit model' : 'Add model'}
                open={open}
                onCancel={() => setOpen(false)}
                onOk={() => void submit()}
                destroyOnHidden
            >
                <Form form={form} layout="vertical" className="mt-4">
                    <Form.Item label="Value" name="value" rules={[{ required: true, message: 'Value is required' }]}>
                        <Input placeholder="claude-sonnet-5" />
                    </Form.Item>
                    <Form.Item label="Name" name="name" rules={[{ required: true, message: 'Name is required' }]}>
                        <Input placeholder="Claude Sonnet 5" />
                    </Form.Item>
                    <Form.Item label="Tag" name="tag" extra="Pick a preset or type your own.">
                        <Select virtual={false} mode="tags" options={tagOptions} placeholder="balanced" />
                    </Form.Item>
                    <Form.Item label="Provider" name="provider">
                        <Input placeholder="Anthropic" />
                    </Form.Item>
                </Form>
            </Modal>
        </Card>
    )
}

// CheckRow renders one doctor check as a Flex row (List is deprecated in
// antd v6) with a themed status icon.
function CheckRow({ name, ok, message }: DoctorCheck) {
    const { token } = theme.useToken()

    return (
        <Flex align="flex-start" gap={8}>
            {ok ? (
                <CheckCircleFilled
                    aria-hidden="true"
                    style={{ color: token.colorSuccess, fontSize: 16, marginTop: 3 }}
                />
            ) : (
                <CloseCircleFilled aria-hidden="true" style={{ color: token.colorError, fontSize: 16, marginTop: 3 }} />
            )}
            <Flex vertical flex={1}>
                <Typography.Text strong>{name}</Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {message}
                </Typography.Text>
            </Flex>
        </Flex>
    )
}

// DoctorTab runs the shared installation checks (GET /api/doctor) on open
// and on demand, rendering them in one card with a pass summary.
function DoctorTab() {
    const dispatch = useAppDispatch()
    const checks = useAppSelector(selectDoctorChecks)
    const running = useAppSelector(selectDoctorRunning)
    const error = useAppSelector(selectDoctorError)

    useEffect(() => {
        void dispatch(runDoctor())
    }, [dispatch])

    const passed = checks?.filter((c) => c.ok).length ?? 0
    const percent = checks && checks.length > 0 ? Math.round((passed / checks.length) * 100) : 0

    return (
        <Card size="small" className="max-w-3xl" title={<CardTitle icon={MedicineBoxOutlined}>Checks</CardTitle>}>
            {error && <Alert type="error" showIcon message={error} className="mb-4" />}
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
            {checks && (
                <>
                    <Flex align="center" gap={12} className="mt-4">
                        <Progress percent={percent} size="small" className="flex-1" />
                        <Typography.Text type="secondary">
                            {passed} of {checks.length} passed
                        </Typography.Text>
                    </Flex>
                    <Divider />
                    <div className="grid gap-4 md:grid-cols-2">
                        {checks.map((c) => (
                            <CheckRow key={c.name} {...c} />
                        ))}
                    </div>
                </>
            )}
        </Card>
    )
}

// GitAccountSection shows the connected identity (avatar, profile link,
// account dates, scopes) with the disconnect action.
function GitAccountSection({ auth, onDisconnect }: { auth: GitHubIdentity; onDisconnect: () => void }) {
    const scopes = auth.scopes
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '')

    return (
        <div className="flex items-center gap-3">
            {auth.avatar_url !== '' && <Avatar src={auth.avatar_url} size={48} />}
            <div className="min-w-0 flex-1">
                <p className="text-xs text-subtle">Connected as</p>
                <p className="mt-1 truncate text-base font-semibold text-heading">
                    {auth.profile_url !== '' ? (
                        <a href={auth.profile_url} target="_blank" rel="noreferrer" className="hover:underline">
                            {auth.display_name || auth.username}
                        </a>
                    ) : (
                        auth.display_name || auth.username
                    )}
                </p>
                <p className="mt-1.5 truncate text-xs text-subtle">{auth.email || auth.username}</p>
                {(auth.account_created_at !== '' || auth.account_updated_at !== '') && (
                    <Flex align="center" gap={4} className="mt-1.5">
                        <ClockCircleOutlined aria-hidden="true" className="text-subtle" />
                        <p className="truncate text-xs text-subtle">
                            Member since {auth.account_created_at.slice(0, 10)}
                            {auth.account_updated_at !== '' && ` · Updated ${auth.account_updated_at.slice(0, 10)}`}
                        </p>
                    </Flex>
                )}
                {scopes.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                        {scopes.map((s) => (
                            <Tag key={s}>{s}</Tag>
                        ))}
                    </div>
                )}
            </div>
            <Button onClick={onDisconnect}>Disconnect</Button>
        </div>
    )
}

// GitRemoteSection holds the sync repo URL input; suggestions come from the
// connected account, and a picked public repo triggers the security warning.
function GitRemoteSection({
    repos,
    selectedRepo,
    onSelect,
    publicSelected
}: {
    repos: GitHubRepo[] | null
    selectedRepo: GitHubRepo | null
    onSelect: (repo: GitHubRepo | null) => void
    publicSelected: boolean
}) {
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

    return (
        <>
            <Form.Item label="Git remote URL" name="repo_url">
                <AutoComplete
                    virtual={false}
                    placeholder="https://github.com/you/wiki.git"
                    options={repoOptions}
                    onSelect={(_value, option) => {
                        const repo = (option as { repo?: GitHubRepo }).repo
                        onSelect(repo ?? null)
                    }}
                    onChange={(value) => {
                        if (selectedRepo && selectedRepo.clone_url !== value) onSelect(null)
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
        </>
    )
}

// GitSyncSection is the single auto-sync switch.
function GitSyncSection() {
    return (
        <Form.Item
            name="sync_enabled"
            valuePropName="checked"
            label="Auto-sync the wiki to the remote"
            extra="Stores your wiki in a remote git repository. Thoth initializes the repo if needed, commits the current tree, and pushes the branch."
        >
            <Switch />
        </Form.Item>
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
            <Card size="small" className="max-w-2xl" title={<CardTitle icon={GithubOutlined}>GitHub</CardTitle>}>
                <SectionHeading icon={GithubOutlined}>Account</SectionHeading>
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
        <Card size="small" className="max-w-3xl" title={<CardTitle icon={GithubOutlined}>GitHub</CardTitle>}>
            <SectionHeading icon={GithubOutlined}>Account</SectionHeading>
            <GitAccountSection auth={auth} onDisconnect={() => void disconnect()} />
            <Divider />
            <div className="grid gap-4 md:grid-cols-2">
                <div>
                    <SectionHeading icon={BranchesOutlined}>Remote</SectionHeading>
                    <GitRemoteSection
                        repos={repos}
                        selectedRepo={selectedRepo}
                        onSelect={setSelectedRepo}
                        publicSelected={publicSelected}
                    />
                </div>
                <div>
                    <SectionHeading icon={SyncOutlined}>Sync</SectionHeading>
                    <GitSyncSection />
                </div>
            </div>
            <Divider />
            {gitError && <Alert type="error" showIcon message={gitError} />}
            <div className="flex justify-end gap-3">
                <Button htmlType="submit">Save</Button>
                <Button type="primary" loading={pushing} disabled={publicSelected} onClick={() => void push()}>
                    Initialize & Push
                </Button>
            </div>
        </Card>
    )
}
