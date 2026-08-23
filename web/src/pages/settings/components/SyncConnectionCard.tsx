import { useEffect, useState } from 'react'
import { Alert, Button, Divider, Flex, Form, Input, Modal, Select, Switch, Tag } from 'antd'
import {
    BranchesOutlined,
    CloudOutlined,
    DeleteOutlined,
    FolderOpenOutlined,
    HistoryOutlined,
    SyncOutlined
} from '@ant-design/icons'
import type { Connection, SyncProvider, SyncSnapshot, SyncTarget } from '../../../api/client'

// SyncConnectionCard is one configured sync destination: identity line,
// editable target fields (from the provider's field descriptors), the
// enabled switch, and push/disconnect actions. Git connections offer a
// repository picker fed by the account's targets; local shows its backup
// folder; s3 its bucket/region/prefix. s3 + local connections can also
// restore the wiki from a stored snapshot (the Restore action).
export function SyncConnectionCard({
    connection,
    provider,
    targets,
    snapshots,
    pushing,
    restoring,
    onPush,
    onRestore,
    onDisconnect,
    onSetActive,
    onUpdate
}: {
    connection: Connection
    provider: SyncProvider
    targets: SyncTarget[] | null
    snapshots: SyncSnapshot[] | null
    pushing: boolean
    restoring: boolean
    onPush: () => void
    onRestore: (key: string) => void
    onDisconnect: () => void
    onSetActive: () => void
    onUpdate: (input: { name?: string; enabled?: boolean; config?: Record<string, string> }) => void
}) {
    const [form] = Form.useForm<{ config: Record<string, string> }>()
    const [restoreOpen, setRestoreOpen] = useState(false)
    const [restoreKey, setRestoreKey] = useState('')

    useEffect(() => {
        // Seed the form with the connection's non-secret config; the git
        // repository selection rides the same form path so it round-trips.
        const cfg = nonSecretConfig(provider, connection)
        if (provider.kind === 'git') cfg.repo_url = stringValue(connection.config.repo_url)
        form.setFieldsValue({ config: cfg })
    }, [connection, provider, form])

    const title =
        provider.name === connection.provider_name
            ? connection.name
            : `${connection.provider_name} · ${connection.name}`
    const icon =
        provider.kind === 'local' ? (
            <FolderOpenOutlined aria-hidden="true" />
        ) : provider.kind === 's3' ? (
            <CloudOutlined aria-hidden="true" />
        ) : (
            <BranchesOutlined aria-hidden="true" />
        )
    const identityLine = connection.identity
        ? connection.identity.display_name || connection.identity.username || connection.identity.account || ''
        : ''

    const nonSecret = provider.fields.filter((f) => !f.secret)
    const isGit = provider.kind === 'git'
    const canRestore = provider.kind === 's3' || provider.kind === 'local'

    return (
        <div className="rounded-lg border border-line bg-surface p-4 shadow-card">
            <Flex align="center" gap={8}>
                <span className="text-subtle">{icon}</span>
                <div className="min-w-0 flex-1">
                    <p className="truncate text-base font-semibold text-heading">{title}</p>
                    <p className="truncate text-xs text-subtle">{identityLine || provider.kind}</p>
                </div>
                {connection.active && <Tag color="blue">Active</Tag>}
                {connection.protected && <Tag color="gold">Protected</Tag>}
                <Switch
                    checked={connection.enabled}
                    aria-label="Enabled"
                    onChange={(enabled) => onUpdate({ enabled })}
                />
            </Flex>

            <Form form={form} layout="vertical" className="mt-4">
                {isGit ? (
                    <Form.Item label="Sync repository" name={['config', 'repo_url']}>
                        <Select
                            virtual={false}
                            placeholder="Choose a repository…"
                            value={stringValue(connection.config.repo_url)}
                            options={(targets ?? []).map((t) => ({ value: t.url, label: t.full_name }))}
                            onChange={(value: string) => onUpdate({ config: { repo_url: value } })}
                        />
                    </Form.Item>
                ) : (
                    nonSecret.map((f) => (
                        <Form.Item key={f.key} label={f.label} name={['config', f.key]} className="mb-2">
                            <Input
                                placeholder={f.label}
                                onBlur={(e) => {
                                    if (e.target.value !== stringValue(connection.config[f.key])) {
                                        onUpdate({ config: { [f.key]: e.target.value } })
                                    }
                                }}
                            />
                        </Form.Item>
                    ))
                )}
            </Form>

            {connection.last_error !== '' && (
                <Alert type="error" showIcon title={connection.last_error} className="mb-3" />
            )}
            {connection.push_history.length > 0 && (
                <div className="mb-3 text-xs text-subtle">
                    <Flex gap={6} align="center" className="mb-1">
                        <HistoryOutlined aria-hidden="true" />
                        <span className="font-medium text-faint">Recent runs</span>
                    </Flex>
                    <ul className="space-y-0.5 pl-1">
                        {connection.push_history.slice(0, 5).map((h, i) => (
                            <li key={i} className="flex items-baseline gap-1.5">
                                <span aria-hidden="true" className={h.ok ? 'text-success' : 'text-error'}>
                                    {h.ok ? '✓' : '✗'}
                                </span>
                                <span className="font-mono">{h.at.slice(0, 16).replace('T', ' ')}</span>
                                {!h.ok && h.error ? <span className="truncate text-subtle">— {h.error}</span> : null}
                            </li>
                        ))}
                    </ul>
                </div>
            )}
            <Divider className="my-3" />
            <Flex gap={8} justify="end" align="center">
                <span className="mr-auto text-xs text-subtle">
                    {connection.last_synced_at !== ''
                        ? `Last synced ${connection.last_synced_at.slice(0, 10)}`
                        : 'Never synced'}
                </span>
                {!connection.active && (
                    <Button size="small" onClick={onSetActive}>
                        Set active
                    </Button>
                )}
                {canRestore && (
                    <Button
                        size="small"
                        icon={<HistoryOutlined aria-hidden="true" />}
                        onClick={() => setRestoreOpen(true)}
                    >
                        Restore
                    </Button>
                )}
                <Button
                    type="primary"
                    size="small"
                    icon={<SyncOutlined aria-hidden="true" />}
                    loading={pushing}
                    onClick={onPush}
                >
                    Push now
                </Button>
                {!connection.protected && (
                    <Button size="small" danger icon={<DeleteOutlined aria-hidden="true" />} onClick={onDisconnect}>
                        Disconnect
                    </Button>
                )}
            </Flex>

            <Modal
                title="Restore the wiki from a snapshot"
                open={restoreOpen}
                onCancel={() => setRestoreOpen(false)}
                onOk={() => {
                    onRestore(restoreKey)
                    setRestoreOpen(false)
                }}
                okText="Restore"
                okButtonProps={{ danger: true, loading: restoring }}
                destroyOnClose
            >
                <p className="mb-3 text-sm text-subtle">
                    The wiki is overwritten by the selected snapshot. A local backup of the current wiki is taken first,
                    so nothing is lost.
                </p>
                {snapshots === null ? (
                    <p className="text-sm text-subtle">Loading snapshots…</p>
                ) : snapshots.length === 0 ? (
                    <p className="text-sm text-subtle">No snapshots stored for this destination yet.</p>
                ) : (
                    <Select
                        virtual={false}
                        className="w-full"
                        placeholder="Choose a snapshot…"
                        value={restoreKey || undefined}
                        options={snapshots.map((s) => ({
                            value: s.key,
                            label: s.time ? `${s.key} (${s.time})` : s.key
                        }))}
                        onChange={(v: string) => setRestoreKey(v)}
                    />
                )}
            </Modal>
        </div>
    )
}

// nonSecretConfig builds the form's initial values from the connection's
// non-secret config fields (secret fields never round-trip).
function nonSecretConfig(provider: SyncProvider, connection: Connection): Record<string, string> {
    const out: Record<string, string> = {}
    for (const f of provider.fields) {
        if (!f.secret) out[f.key] = stringValue(connection.config[f.key])
    }
    return out
}

// stringValue narrows a config value (string | boolean on the wire) to a
// string, the shape the editable fields use.
function stringValue(v: unknown): string {
    return typeof v === 'string' ? v : ''
}
