import { useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import {
    Alert,
    App,
    AutoComplete,
    Avatar,
    Button,
    Card,
    Collapse,
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
import type { FormInstance, TableProps } from 'antd'
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
import type { DoctorCheck, GitHubIdentity, GitHubRepo, LLMModel, ModelInput, Settings } from '../../api/client'
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
} from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { AppHeader } from '../../shared/AppHeader'
import { WikiPathInput } from '../../shared/WikiPathInput'
import { navigateSegment, useViewRoute } from '../../hooks/useView'

type Tab = 'general' | 'providers' | 'git' | 'doctor'

const tabs: { id: Tab; label: string; icon: typeof SettingsIcon }[] = [
    { id: 'general', label: 'General', icon: SettingsIcon },
    { id: 'providers', label: 'Providers', icon: ApiOutlined },
    { id: 'git', label: 'Git remote', icon: BranchesOutlined },
    { id: 'doctor', label: 'Doctor', icon: MedicineBoxOutlined }
]

export function SettingsPage() {
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
                <Form
                    form={form}
                    layout="vertical"
                    onFinish={(values) => void save(values)}
                    className="flex min-h-0 flex-1 flex-col"
                >
                    <Tabs
                        tabPlacement="start"
                        activeKey={tab}
                        // The rail is styled by the Tabs tokens in theme.ts
                        // plus the .settings-tabs pill rules in index.css.
                        className="min-h-0 flex-1"
                        classNames={{ root: 'settings-tabs' }}
                        tabBarStyle={{ minWidth: 140 }}
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
                                ) : t.id === 'providers' ? (
                                    <ProvidersTab status={status} />
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

// SectionHeading is the icon'd micro-title that opens each card section
// (the DashboardPage kicker pattern, inside a Card).
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

// The scaffold folder set offered as suggestions in the General tab; the
// server keeps these defaults whenever wiki_folders is unset.
const defaultFolders = ['inbox', 'meetings', 'projects', 'links', 'setup', 'knowledge', 'todos', 'daily']

// GeneralTab is the wiki path + provider/model pickers in one card, with the
// save button and the saved/error feedback under them. Credentials and the
// model registry live on the Providers tab.
function GeneralTab({ status }: { status: 'idle' | 'saved' | 'error' }) {
    const settings = useAppSelector(selectSettings)
    const groups = useAppSelector(selectModelGroups)
    const form = Form.useFormInstance<Settings>()
    // The server reports the wiki default for the mode it runs in; the
    // fallback covers the render before the first health response lands.
    const defaultWiki = useAppSelector(selectHealth)?.default_wiki_path ?? '~/.thoth/wiki'

    // The model picker is a two-field cascade: the Provider select is view
    // state (which provider's models the Model select shows), the Model
    // select is the form field that gets saved.
    const [provider, setProvider] = useState<string | null>(null)

    // Resolve the saved model's provider once settings arrive, so an already
    // selected model stays visible in the cascade.
    useEffect(() => {
        if (!settings.data || settings.data.model === '') return
        const g = groups.find((gr) => gr.models.some((m) => m.value === settings.data!.model))
        if (g) setProvider(g.provider)
    }, [settings.data, groups])

    // Provider options are the registry's provider labels (A→Z, as the
    // server groups them); the current provider's models become the Model
    // select's options, with name + tag so optionRender can show the tag.
    const providerOptions = useMemo(
        () => groups.filter((g) => g.provider !== '').map((g) => ({ label: g.provider, value: g.provider })),
        [groups]
    )
    const modelOptions = useMemo(() => {
        const g = groups.find((gr) => gr.provider === provider)
        return (g?.models ?? []).map((m) => ({ label: m.name, value: m.value, tag: m.tag }))
    }, [groups, provider])

    const onProviderChange = (value: string) => {
        setProvider(value)
        // A model from the previous provider no longer applies; clear it so
        // the user picks one from the newly selected provider.
        form.setFieldValue('model', undefined)
    }

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
                <Form.Item
                    label="Provider"
                    htmlFor="settings-provider"
                    extra="The provider your selected model comes from."
                >
                    <Select
                        id="settings-provider"
                        virtual={false}
                        placeholder="Select a provider"
                        options={providerOptions}
                        notFoundContent="No models — add a provider model in the Providers tab"
                        value={provider ?? undefined}
                        onChange={onProviderChange}
                    />
                </Form.Item>
                <Form.Item label="Model" name="model" extra="Applied to all chats after the app restarts.">
                    <Select
                        virtual={false}
                        options={modelOptions}
                        notFoundContent="Select a provider first"
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
            </div>
            <Form.Item
                label="Scaffold folders"
                name="wiki_folders"
                extra="The folders created when a wiki is scaffolded. Type your own set or keep the defaults."
            >
                <Select
                    virtual={false}
                    mode="tags"
                    placeholder="inbox, meetings, projects, …"
                    options={defaultFolders.map((f) => ({ value: f }))}
                    tokenSeparators={[',']}
                />
            </Form.Item>
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

// ProvidersTab is the provider management surface: a fallback API key for
// providers without a key of their own, then one collapsible panel per
// provider holding its credential form (base URL + API key) and the models
// registered under it. Every model mutation refetches the registry (the
// server re-groups and re-sorts) and settings (a rename may have moved the
// selected-model setting, a delete may have cleared it).
function ProvidersTab({ status }: { status: 'idle' | 'saved' | 'error' }) {
    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)
    const models = useAppSelector(selectModelList)
    const groups = useAppSelector(selectModelGroups)
    const { message } = App.useApp()
    const [open, setOpen] = useState(false)
    const [editing, setEditing] = useState<LLMModel | null>(null)
    const [modelForm] = Form.useForm<ModelInput>()

    // Panels are the union of registry providers (the model groups, already
    // A→Z) and any leftover settings keys, so a provider keeps its panel even
    // when its last model is deleted.
    const providerNames = useMemo(() => {
        const names = groups.map((g) => g.provider)
        for (const p of Object.keys(settings.data?.providers ?? {})) {
            if (!names.includes(p)) names.push(p)
        }
        return names
    }, [groups, settings.data])

    // The tag dropdown offers every tag already in the registry (sorted);
    // mode="tags" also lets the user type a new one.
    const tagOptions = useMemo(
        () =>
            [...new Set(groups.flatMap((g) => g.models.map((m) => m.tag)).filter((t) => t !== ''))]
                .sort()
                .map((t) => ({ value: t })),
        [groups]
    )

    const openAdd = (provider: string) => {
        setEditing(null)
        modelForm.resetFields()
        modelForm.setFieldsValue({ provider })
        setOpen(true)
    }

    const openEdit = (m: LLMModel) => {
        setEditing(m)
        modelForm.setFieldsValue(m)
        setOpen(true)
    }

    const submit = async () => {
        const values = await modelForm.validateFields()
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

    // The per-provider table is the flat registry table without the (now
    // redundant) provider column; the panel header names the provider.
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
        <Card size="small" className="max-w-4xl" title={<CardTitle icon={ApiOutlined}>Providers</CardTitle>}>
            {providerNames.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No models yet">
                    <Button type="primary" onClick={() => openAdd('')}>
                        Add your first model
                    </Button>
                </Empty>
            ) : (
                <Collapse
                    defaultActiveKey={[providerNames[0]!]}
                    items={providerNames.map((provider) => ({
                        key: provider,
                        label: (
                            <ProviderHeader
                                provider={provider}
                                modelCount={models.filter((m) => m.provider === provider).length}
                                hasKey={settings.data?.providers?.[provider]?.has_api_key === true}
                                baseURL={settings.data?.providers?.[provider]?.base_url ?? ''}
                            />
                        ),
                        children: (
                            <ProviderPanel
                                provider={provider}
                                models={models.filter((m) => m.provider === provider)}
                                columns={columns}
                                onAdd={() => openAdd(provider)}
                            />
                        )
                    }))}
                />
            )}
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
            <ModelModal
                open={open}
                editing={editing}
                form={modelForm}
                tagOptions={tagOptions}
                onCancel={() => setOpen(false)}
                onOk={() => void submit()}
            />
        </Card>
    )
}

// ProviderHeader is a Collapse panel title: the provider name plus status
// chips (model count, key state, endpoint).
function ProviderHeader({
    provider,
    modelCount,
    hasKey,
    baseURL
}: {
    provider: string
    modelCount: number
    hasKey: boolean
    baseURL: string
}) {
    return (
        <Flex align="center" justify="space-between" gap={8} className="pr-2">
            <span className="font-medium text-heading">{provider === '' ? 'Unassigned' : provider}</span>
            <Flex align="center" gap={6}>
                <Tag>
                    {modelCount} model{modelCount === 1 ? '' : 's'}
                </Tag>
                {hasKey ? <Tag color="success">key set</Tag> : <Tag>no key</Tag>}
                {baseURL !== '' ? <Tag>custom endpoint</Tag> : <Tag>default endpoint</Tag>}
            </Flex>
        </Flex>
    )
}

// ProviderKeyField is a provider's API key input; like the shared key it is
// write-only, so an empty value leaves the stored key untouched.
function ProviderKeyField({ provider }: { provider: string }) {
    const settings = useAppSelector(selectSettings)
    const hasKey = settings.data?.providers?.[provider]?.has_api_key === true

    return (
        <Form.Item
            label={
                <Flex align="center" gap={6}>
                    API key
                    {hasKey ? <Tag color="success">Configured</Tag> : <Tag>Not set</Tag>}
                </Flex>
            }
            name={['providers', provider, 'api_key']}
            extra={hasKey ? 'A key is saved — leave blank to keep it.' : 'Falls back to the shared API key when blank.'}
        >
            <Input.Password placeholder={hasKey ? '•••••••• (saved)' : 'sk-…'} autoComplete="off" />
        </Form.Item>
    )
}

// ProviderPanel is one Collapse body: the credential form for a named
// provider (omitted for the Unassigned catch-all) plus its registered models
// table.
function ProviderPanel({
    provider,
    models,
    columns,
    onAdd
}: {
    provider: string
    models: LLMModel[]
    columns: TableProps<LLMModel>['columns']
    onAdd: () => void
}) {
    return (
        <div className="grid gap-4">
            {provider !== '' && (
                <div className="grid gap-3 md:grid-cols-2">
                    <Form.Item
                        label="Base URL"
                        name={['providers', provider, 'base_url']}
                        extra="Empty uses the provider's default endpoint."
                    >
                        <Input placeholder="https://api.example.com" autoComplete="off" />
                    </Form.Item>
                    <ProviderKeyField provider={provider} />
                </div>
            )}
            <Divider />
            <Flex align="center" justify="space-between" className="mb-3">
                <SectionHeading icon={ApiOutlined}>Models</SectionHeading>
                <Button size="small" icon={<ApiOutlined aria-hidden="true" />} onClick={onAdd}>
                    Add model
                </Button>
            </Flex>
            {models.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No models for this provider yet" />
            ) : (
                <Table<LLMModel> rowKey="id" size="small" columns={columns} dataSource={models} pagination={false} />
            )}
        </div>
    )
}

// ModelModal is the shared add/edit form for an LLM model; the provider field
// is pre-filled when the modal opens from a provider panel.
function ModelModal({
    open,
    editing,
    form,
    tagOptions,
    onCancel,
    onOk
}: {
    open: boolean
    editing: LLMModel | null
    form: FormInstance<ModelInput>
    tagOptions: { value: string }[]
    onCancel: () => void
    onOk: () => void
}) {
    return (
        <Modal
            title={editing ? 'Edit model' : 'Add model'}
            open={open}
            onCancel={onCancel}
            onOk={() => void onOk()}
            destroyOnHidden
        >
            <Form form={form} layout="vertical" className="mt-4">
                <Form.Item label="Value" name="value" rules={[{ required: true, message: 'Value is required' }]}>
                    <Input placeholder="my-model" />
                </Form.Item>
                <Form.Item label="Name" name="name" rules={[{ required: true, message: 'Name is required' }]}>
                    <Input placeholder="My Model" />
                </Form.Item>
                <Form.Item label="Tag" name="tag" extra="Pick a preset or type your own.">
                    <Select virtual={false} mode="tags" options={tagOptions} placeholder="balanced" />
                </Form.Item>
                <Form.Item label="Provider" name="provider">
                    <Input placeholder="Anthropic" />
                </Form.Item>
            </Form>
        </Modal>
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
