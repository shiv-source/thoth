import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Form, Input, Select } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import type { SyncProvider } from '../../../api/client'

// SyncConnectForm is the "connect a destination" card: pick a provider, fill
// its credential/target fields (driven by the provider's field descriptors —
// secrets render as password inputs), name the connection, and connect. The
// server verifies credentials before storing. Fields fill the panel width in
// pairs so the form reads as a whole, with the action tucked at the end.
export function SyncConnectForm({
    providers,
    connecting,
    error,
    onConnect
}: {
    providers: SyncProvider[]
    connecting: boolean
    error: string | null
    onConnect: (input: { provider_id: number; name: string; config: Record<string, string> }) => void
}) {
    const [form] = Form.useForm<{ name: string; provider_id: number } & Record<string, string>>()
    const [providerId, setProviderId] = useState<number | null>(null)
    const provider = useMemo(() => providers.find((p) => p.id === providerId) ?? null, [providers, providerId])

    useEffect(() => {
        // A provider switch clears the previous provider's credential fields.
        const clear: Record<string, string> = {}
        for (const f of provider?.fields ?? []) clear[f.key] = ''
        form.setFieldsValue(clear)
    }, [provider, form])

    return (
        <div className="rounded-lg border border-line bg-raised p-5">
            <Form
                form={form}
                name="sync-connect"
                layout="vertical"
                onFinish={(values) => {
                    const config: Record<string, string> = {}
                    for (const f of provider?.fields ?? []) config[f.key] = values[f.key] ?? ''
                    onConnect({ provider_id: values.provider_id, name: values.name, config })
                }}
            >
                <div className="grid gap-4 md:grid-cols-2">
                    <Form.Item
                        label="Provider"
                        name="provider_id"
                        rules={[{ required: true, message: 'Choose a provider' }]}
                        className="min-w-0"
                    >
                        <Select
                            virtual={false}
                            placeholder="Choose a destination provider"
                            options={providers.map((p) => ({ value: p.id, label: p.name }))}
                            onChange={(v: number) => setProviderId(v)}
                        />
                    </Form.Item>
                    <Form.Item
                        label="Name"
                        name="name"
                        rules={[{ required: true, message: 'Name the connection' }]}
                        className="min-w-0"
                    >
                        <Input placeholder="home wiki, work, …" />
                    </Form.Item>
                </div>
                {provider && provider.fields.length > 0 && (
                    <div className="grid gap-4 md:grid-cols-2">
                        {provider.fields.map((f) => (
                            <Form.Item
                                key={f.key}
                                label={f.label}
                                name={f.key}
                                rules={f.required ? [{ required: true, message: `${f.label} is required` }] : []}
                                className="min-w-0"
                            >
                                {f.secret ? <Input.Password placeholder={f.label} /> : <Input placeholder={f.label} />}
                            </Form.Item>
                        ))}
                    </div>
                )}
                {error && <Alert type="error" showIcon title={error} className="mb-4" />}
                <div className="flex justify-end">
                    <Button
                        type="primary"
                        icon={<PlusOutlined aria-hidden="true" />}
                        htmlType="submit"
                        loading={connecting}
                    >
                        Connect
                    </Button>
                </div>
            </Form>
        </div>
    )
}
