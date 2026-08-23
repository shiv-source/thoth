import type { FormInstance } from 'antd'
import { AutoComplete, Form, Input, Modal, Select } from 'antd'
import type { LLMModel, ModelInput, Provider } from '../../../api/client'

// ModelModal is the shared add/edit form for an LLM model. The provider
// select is pre-filled with the panel's provider when the modal opens from a
// provider; "Unassigned" covers models without a provider.
export function ModelModal({
    open,
    editing,
    form,
    providers,
    tagOptions,
    onCancel,
    onOk
}: {
    open: boolean
    editing: LLMModel | null
    form: FormInstance<ModelInput>
    providers: Provider[]
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
                    <AutoComplete options={tagOptions} placeholder="balanced" />
                </Form.Item>
                <Form.Item label="Provider" name="provider_id" extra="The provider this model belongs to.">
                    <Select
                        virtual={false}
                        placeholder="Select a provider"
                        options={[
                            ...providers.map((p) => ({ label: p.name, value: p.id })),
                            { label: 'Unassigned', value: 0 }
                        ]}
                    />
                </Form.Item>
            </Form>
        </Modal>
    )
}
