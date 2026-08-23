import type { FormInstance } from 'antd'
import { Form, Input, Modal } from 'antd'
import type { Provider, ProviderInput } from '../../../api/client'

// ProviderModal is the add/edit form for a provider: its name, base URL
// override, and write-only API key. On edit an empty key leaves the stored
// key untouched (the server never echoes it back).
export function ProviderModal({
    open,
    editing,
    form,
    onCancel,
    onOk
}: {
    open: boolean
    editing: Provider | null
    form: FormInstance<ProviderInput>
    onCancel: () => void
    onOk: () => void
}) {
    return (
        <Modal
            title={editing ? 'Edit provider' : 'Add provider'}
            open={open}
            onCancel={onCancel}
            onOk={() => void onOk()}
            destroyOnHidden
        >
            <Form form={form} layout="vertical" className="mt-4">
                <Form.Item label="Name" name="name" rules={[{ required: true, message: 'Name is required' }]}>
                    <Input placeholder="Anthropic" autoComplete="off" />
                </Form.Item>
                <Form.Item label="Base URL" name="base_url" extra="Empty uses the provider's default endpoint.">
                    <Input placeholder="https://api.example.com" autoComplete="off" />
                </Form.Item>
                <Form.Item
                    label={editing ? 'API key (leave blank to keep)' : 'API key'}
                    name="api_key"
                    extra="Stored locally in thoth.db — never shown again."
                >
                    <Input.Password placeholder={editing ? '•••••••• (saved)' : 'sk-…'} autoComplete="off" />
                </Form.Item>
            </Form>
        </Modal>
    )
}
