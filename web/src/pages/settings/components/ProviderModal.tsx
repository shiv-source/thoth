import type { FormInstance } from 'antd'
import { Button, Form, Input, Modal } from 'antd'
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons'
import type { Provider } from '../../../api/client'

// ProviderFormValues is the add/edit form's shape. custom_headers is edited as
// a list of name/value rows; the page reduces it to an object on submit.
export interface ProviderFormValues {
    name: string
    base_url?: string
    api_key?: string
    custom_headers?: { name: string; value: string }[]
}

// ProviderModal is the add/edit form for a provider: its name, base URL
// override, write-only API key, and custom request headers (e.g. Portkey's
// x-portkey-* routing headers) as key/value rows. On edit an empty key leaves
// the stored key untouched (the server never echoes it back).
export function ProviderModal({
    open,
    editing,
    form,
    onCancel,
    onOk
}: {
    open: boolean
    editing: Provider | null
    form: FormInstance<ProviderFormValues>
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
                <Form.Item
                    label="Custom headers"
                    extra="Extra HTTP headers sent on every request — used by gateways like Portkey (e.g. x-portkey-provider)."
                >
                    <Form.List name="custom_headers">
                        {(fields, { add, remove }) => (
                            <div className="space-y-2">
                                {fields.map((field) => (
                                    <div key={field.key} className="flex items-center gap-2">
                                        <Form.Item name={[field.name, 'name']} noStyle>
                                            <Input
                                                placeholder="x-portkey-provider"
                                                autoComplete="off"
                                                className="w-1/2"
                                            />
                                        </Form.Item>
                                        <Form.Item name={[field.name, 'value']} noStyle>
                                            <Input placeholder="anthropic" autoComplete="off" className="flex-1" />
                                        </Form.Item>
                                        <Button
                                            type="text"
                                            danger
                                            aria-label="Remove header"
                                            icon={<MinusCircleOutlined aria-hidden="true" />}
                                            onClick={() => remove(field.name)}
                                        />
                                    </div>
                                ))}
                                <Button
                                    type="dashed"
                                    block
                                    icon={<PlusOutlined aria-hidden="true" />}
                                    onClick={() => add({ name: '', value: '' })}
                                >
                                    Add header
                                </Button>
                            </div>
                        )}
                    </Form.List>
                </Form.Item>
            </Form>
        </Modal>
    )
}
