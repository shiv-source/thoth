import { useState } from 'react'
import { Alert, Button, Flex, Form, Input, Select, Tag } from 'antd'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import type { SyncProvider } from '../../../api/client'

const KNOWN_DRIVERS = [
    { value: 'github', label: 'GitHub' },
    { value: 'gitlab', label: 'GitLab' },
    { value: 's3', label: 'AWS S3 / S3-compatible' },
    { value: 'local', label: 'Local backup' }
]

// SyncProviderEditor manages the sync provider catalog: the built-ins plus
// user-added rows (a known driver with a custom name / base URL). Protected
// rows (the local backup) are locked.
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
            <Flex gap={8} wrap="wrap">
                {providers.map((p) => (
                    <div
                        key={p.id}
                        className="flex items-center gap-2 rounded-md border border-line bg-raised px-3 py-2"
                    >
                        <span className="text-sm font-medium text-heading">{p.name}</span>
                        <Tag>{p.kind}</Tag>
                        {p.protected && <Tag color="gold">Protected</Tag>}
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
            </Flex>

            <Form
                form={form}
                name="sync-provider-editor"
                layout="inline"
                className="mt-4"
                onFinish={(values) => {
                    setError(null)
                    onCreate({ name: values.name, driver: values.driver, base_url: values.base_url })
                    form.resetFields()
                }}
            >
                <Form.Item name="name" label="Provider name" rules={[{ required: true, message: 'Name' }]}>
                    <Input placeholder="Provider name (e.g. GitHub Enterprise)" />
                </Form.Item>
                <Form.Item name="driver" label="Driver" rules={[{ required: true, message: 'Driver' }]}>
                    <Select virtual={false} placeholder="Driver" options={KNOWN_DRIVERS} style={{ minWidth: 180 }} />
                </Form.Item>
                <Form.Item name="base_url" label="Base URL">
                    <Input placeholder="Base URL (optional)" style={{ minWidth: 220 }} />
                </Form.Item>
                <Button type="dashed" icon={<PlusOutlined aria-hidden="true" />} htmlType="submit">
                    Add provider
                </Button>
            </Form>
            {error && <Alert type="error" showIcon title={error} className="mt-3" />}
        </div>
    )
}
