import { useEffect } from 'react'
import { App, Alert, Card, Collapse, Divider, Empty, Flex, Spin } from 'antd'
import { ApiOutlined, CloudServerOutlined } from '@ant-design/icons'
import type { Connection, SyncProvider } from '../../api/client'
import {
    connectSync,
    createSyncProvider,
    deleteSyncProvider,
    disconnectSync,
    fetchSync,
    fetchSyncSnapshots,
    fetchSyncTargets,
    pushSync,
    restoreSync,
    selectSyncConnecting,
    selectSyncConnections,
    selectSyncError,
    selectSyncLoading,
    selectSyncProviders,
    selectSyncPushing,
    selectSyncRestoring,
    selectSyncSnapshots,
    selectSyncTargets,
    setActiveSync,
    updateSyncConnection
} from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CardTitle } from './components/CardTitle'
import { SectionHeading } from './components/SectionHeading'
import { SyncConnectForm } from './components/SyncConnectForm'
import { SyncConnectionCard } from './components/SyncConnectionCard'
import { SyncProviderEditor } from './components/SyncProviderEditor'
import { SettingsShell } from './SettingsShell'

// ConnectionRow renders one connection card, owning its own targets selector
// (hooks cannot run inside a map).
function ConnectionRow({
    connection,
    provider,
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
    pushing: boolean
    restoring: boolean
    onPush: () => void
    onRestore: (key: string) => void
    onDisconnect: () => void
    onSetActive: () => void
    onUpdate: (input: { name?: string; enabled?: boolean; config?: Record<string, string> }) => void
}) {
    const targets = useAppSelector(selectSyncTargets(connection.id))
    const snapshots = useAppSelector(selectSyncSnapshots(connection.id))
    return (
        <SyncConnectionCard
            connection={connection}
            provider={provider}
            targets={targets}
            snapshots={snapshots}
            pushing={pushing}
            restoring={restoring}
            onPush={onPush}
            onRestore={onRestore}
            onDisconnect={onDisconnect}
            onSetActive={onSetActive}
            onUpdate={onUpdate}
        />
    )
}

// SettingsSyncPage is the multi-provider sync page: the protected local
// backup plus any connected destinations (GitHub/GitLab remotes, S3 buckets),
// the connect flow driven by each provider's field descriptors, and the
// provider catalog editor. S3 + local connections offer a Restore action that
// imports a stored snapshot onto the wiki.
export function SettingsSyncPage() {
    const dispatch = useAppDispatch()
    const { message } = App.useApp()
    const providers = useAppSelector(selectSyncProviders)
    const connections = useAppSelector(selectSyncConnections)
    const loading = useAppSelector(selectSyncLoading)
    const connecting = useAppSelector(selectSyncConnecting)
    const pushing = useAppSelector(selectSyncPushing)
    const restoring = useAppSelector(selectSyncRestoring)
    const error = useAppSelector(selectSyncError)

    useEffect(() => {
        void dispatch(fetchSync())
    }, [dispatch])

    // Git-kind connections offer a repository picker fed by the account's
    // targets; fetch them lazily per connected git connection. The effect
    // keys on the store arrays (stable references) so the fetch runs once per
    // connection list change — never per render.
    const gitConnectionIds = connections
        .filter((c) => providers.find((p) => p.id === c.provider_id)?.kind === 'git')
        .map((c) => c.id)
    useEffect(() => {
        for (const id of gitConnectionIds) void dispatch(fetchSyncTargets(id))
    }, [connections, providers, dispatch])

    // s3 + local connections offer restore; fetch their snapshots lazily the
    // same way, so the restore modal's picker is ready when it opens.
    const restorableConnectionIds = connections
        .filter((c) => {
            const kind = providers.find((p) => p.id === c.provider_id)?.kind
            return kind === 's3' || kind === 'local'
        })
        .map((c) => c.id)
    useEffect(() => {
        for (const id of restorableConnectionIds) void dispatch(fetchSyncSnapshots(id))
    }, [connections, providers, dispatch])

    const connect = async (input: { provider_id: number; name: string; config: Record<string, string> }) => {
        try {
            await dispatch(connectSync(input)).unwrap()
            void message.success('Destination connected')
        } catch (e) {
            void message.error(e instanceof Error ? e.message : 'Could not connect')
        }
    }

    const push = async (id: number) => {
        try {
            const res = await dispatch(pushSync(id)).unwrap()
            void (res.ok ? message.success('Wiki pushed') : message.error(res.error ?? 'Could not push the wiki'))
            void dispatch(fetchSync())
        } catch {
            void message.error('Could not push the wiki')
        }
    }

    const restore = async (id: number, key: string) => {
        try {
            const res = await dispatch(restoreSync({ id, key })).unwrap()
            void message.success(`Wiki restored (${res.files} files)`)
            void dispatch(fetchSync())
        } catch (e) {
            void message.error(e instanceof Error ? e.message : 'Could not restore the wiki')
        }
    }

    const providerById = (id: number) => providers.find((p) => p.id === id)

    return (
        <SettingsShell active="sync">
            <Card title={<CardTitle icon={CloudServerOutlined}>Sync destinations</CardTitle>}>
                <SectionHeading icon={ApiOutlined}>Connected destinations</SectionHeading>
                {loading ? (
                    <Flex justify="center" className="py-10">
                        <Spin />
                    </Flex>
                ) : connections.length === 0 ? (
                    <Empty description="No destinations connected yet — connect one below" />
                ) : (
                    <div className="grid gap-4">
                        {connections.map((c) => {
                            const provider = providerById(c.provider_id)
                            if (!provider) return null
                            return (
                                <ConnectionRow
                                    key={c.id}
                                    connection={c}
                                    provider={provider}
                                    pushing={pushing}
                                    restoring={restoring}
                                    onPush={() => void push(c.id)}
                                    onRestore={(key) => void restore(c.id, key)}
                                    onDisconnect={() => {
                                        void dispatch(disconnectSync(c.id)).then(
                                            () => void message.success('Disconnected')
                                        )
                                    }}
                                    onSetActive={() => void dispatch(setActiveSync(c.id))}
                                    onUpdate={(input) => void dispatch(updateSyncConnection({ id: c.id, input }))}
                                />
                            )
                        })}
                    </div>
                )}

                <Divider />
                <SectionHeading icon={ApiOutlined}>Connect a destination</SectionHeading>
                <SyncConnectForm
                    providers={providers}
                    connecting={connecting}
                    error={error}
                    onConnect={(i) => void connect(i)}
                />

                <Divider />
                <SectionHeading icon={ApiOutlined}>Sync providers</SectionHeading>
                <Collapse
                    ghost
                    items={[
                        {
                            key: 'providers',
                            label: 'Manage the provider catalog (built-ins + your custom providers)',
                            children: (
                                <SyncProviderEditor
                                    providers={providers}
                                    onCreate={(input) =>
                                        void dispatch(createSyncProvider(input)).catch(
                                            () => void message.error('Could not add the provider')
                                        )
                                    }
                                    onDelete={(id) => void dispatch(deleteSyncProvider(id))}
                                />
                            )
                        }
                    ]}
                />

                {error && <Alert type="error" showIcon title={error} className="mt-4" />}
            </Card>
        </SettingsShell>
    )
}
