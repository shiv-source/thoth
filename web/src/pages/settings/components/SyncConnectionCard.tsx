import { useEffect } from 'react'
import { Alert, Button, Divider, Flex, Form, Input, Select, Switch, Tag } from 'antd'
import { BranchesOutlined, CloudOutlined, DeleteOutlined, FolderOpenOutlined, SyncOutlined } from '@ant-design/icons'
import type { Connection, SyncProvider, SyncTarget } from '../../../api/client'

// SyncConnectionCard is one configured sync destination: identity line,
// editable target fields (from the provider's field descriptors), the
// enabled switch, and push/disconnect actions. Git connections offer a
// repository picker fed by the account's targets; local shows its backup
// folder; s3 its bucket/region/prefix.
export function SyncConnectionCard({
    connection,
    provider,
    targets,
    pushing,
    onPush,
    onDisconnect,
    onSetActive,
    onUpdate
}: {
    connection: Connection
    provider: SyncProvider
    targets: SyncTarget[] | null
    pushing: boolean
    onPush: () => void
    onDisconnect: () => void
    onSetActive: () => void
    onUpdate: (input: { name?: string; enabled?: boolean; config?: Record<string, string> }) => void
}) {
    const [form] = Form.useForm<{ config: Record<string, string> }>()

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

    return (
        <div className="rounded-md border border-line bg-raised p-4">
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
