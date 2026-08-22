import type { FormInstance } from 'antd'
import { AutoComplete, Form, Input, Modal } from 'antd'
import type { LLMModel, ModelInput } from '../../../api/client'

// ModelModal is the shared add/edit form for an LLM model; the provider field
// is pre-filled when the modal opens from a provider panel.
export function ModelModal({
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
                    <AutoComplete options={tagOptions} placeholder="balanced" />
                </Form.Item>
                <Form.Item label="Provider" name="provider">
                    <Input placeholder="Anthropic" />
                </Form.Item>
            </Form>
        </Modal>
    )
}
