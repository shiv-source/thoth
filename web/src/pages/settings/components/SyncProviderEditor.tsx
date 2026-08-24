import { useState } from 'react'
import { Alert, Button, Form, Input, Select, Tag, Typography } from 'antd'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import type { SyncProvider } from '../../../api/client'

const KNOWN_DRIVERS = [
    { value: 'github', label: 'GitHub' },
    { value: 'gitlab', label: 'GitLab' },
    { value: 's3', label: 'AWS S3 / S3-compatible' },
    { value: 'local', label: 'Local backup' }
]

// driverLabel maps a driver key to its human name (built-ins only; custom
// rows keep their raw driver key).
function driverLabel(driver: string): string {
    return KNOWN_DRIVERS.find((d) => d.value === driver)?.label ?? driver
}

// SyncProviderEditor manages the sync provider catalog: the built-ins plus
// user-added rows (a known driver with a custom name / base URL). Protected
// rows (the local backup) are locked. The catalog renders as a table so the
// list reads at a glance; the add form sits in its own panel below.
export function SyncProviderEditor({
    providers,
    onCreate,
    onDelete
}: {
    providers: SyncProvider[]
    onCreate: (input: { name: string; driver: string; base_url?: string }) => void
    onDelete: (id: number) => void
}) {
    const [form] = Form.useForm<{ name: string; driver: string; base_url?: string }>()
    const [error, setError] = useState<string | null>(null)

    return (
        <div>
            <div className="rounded-lg border border-line bg-surface p-4">
                <p className="mb-3 text-sm font-medium text-heading">
                    Manage the provider catalog (built-ins + your custom providers)
                </p>
                {providers.length === 0 ? (
                    <Typography.Text type="secondary">No sync providers yet — add one below.</Typography.Text>
                ) : (
                    <div className="divide-y divide-line-soft">
                        {providers.map((p) => (
                            <div key={p.id} className="flex items-center gap-3 py-2.5">
                                <div className="flex min-w-0 flex-1 items-center gap-2">
                                    <span className="truncate font-medium text-heading">{p.name}</span>
                                    {p.base_url !== '' && (
                                        <span className="truncate font-mono text-xs text-subtle">{p.base_url}</span>
                                    )}
                                </div>
                                <span className="w-40 shrink-0 text-sm text-subtle">{driverLabel(p.driver)}</span>
                                <span className="w-20 shrink-0 text-right text-sm text-subtle">
                                    {p.connection_count === 0 ? '—' : p.connection_count}
                                </span>
                                {p.protected ? <Tag color="gold">Protected</Tag> : <Tag>Active</Tag>}
                                <Button
                                    size="small"
                                    type="text"
                                    danger
                                    icon={<DeleteOutlined aria-hidden="true" />}
                                    disabled={p.protected}
                                    aria-label={`Delete ${p.name}`}
                                    onClick={() => onDelete(p.id)}
                                />
                            </div>
                        ))}
                    </div>
                )}
            </div>

            <div className="mt-5 rounded-lg border border-line bg-raised p-4">
                <p className="mb-3 text-sm font-medium text-heading">Add a sync provider</p>
                <Form
                    form={form}
                    name="sync-provider-editor"
                    layout="vertical"
                    onFinish={(values) => {
                        setError(null)
                        onCreate({ name: values.name, driver: values.driver, base_url: values.base_url })
                        form.resetFields()
                    }}
                >
                    <div className="grid gap-4 md:grid-cols-[1fr_200px_1fr]">
                        <Form.Item
                            name="name"
                            label="Provider name"
                            rules={[{ required: true, message: 'Name' }]}
                            className="min-w-0"
                        >
                            <Input placeholder="Provider name (e.g. GitHub Enterprise)" />
                        </Form.Item>
                        <Form.Item
                            name="driver"
                            label="Driver"
                            rules={[{ required: true, message: 'Driver' }]}
                            className="min-w-0"
                        >
                            <Select virtual={false} placeholder="Driver" options={KNOWN_DRIVERS} />
                        </Form.Item>
                        <Form.Item name="base_url" label="Base URL" className="min-w-0">
                            <Input placeholder="Base URL (optional)" />
                        </Form.Item>
                    </div>
                    <div className="flex justify-end">
                        <Button type="primary" icon={<PlusOutlined aria-hidden="true" />} htmlType="submit">
                            Add provider
                        </Button>
                    </div>
                </Form>
            </div>
            {error && <Alert type="error" showIcon title={error} className="mt-3!" />}
        </div>
    )
}
