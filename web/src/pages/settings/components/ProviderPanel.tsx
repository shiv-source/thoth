import type { TableProps } from 'antd'
import { Button, Divider, Empty, Flex, Form, Input, Table } from 'antd'
import { ApiOutlined, LockOutlined } from '@ant-design/icons'
import type { LLMModel } from '../../../api/client'
import { ProviderKeyField } from './ProviderKeyField'
import { SaveFooter } from './SaveFooter'
import { SectionHeading } from './SectionHeading'

// ProviderPanel is one Collapse body: the credential form for a named
// provider (omitted for the Unassigned catch-all) with its own save button,
// plus the provider's registered models table.
export function ProviderPanel({
    provider,
    models,
    columns,
    status,
    saving,
    hasError,
    onAdd
}: {
    provider: string
    models: LLMModel[]
    columns: TableProps<LLMModel>['columns']
    status: 'idle' | 'saved' | 'error'
    saving: boolean
    hasError: boolean
    onAdd: () => void
}) {
    return (
        <div className="grid gap-4">
            {provider !== '' && (
                <>
                    <div className="rounded-md border border-line bg-raised p-4">
                        <SectionHeading icon={LockOutlined}>Credentials</SectionHeading>
                        <div className="grid gap-3 md:grid-cols-2">
                            <Form.Item
                                label="Base URL"
                                name={['providers', provider, 'base_url']}
                                extra="Empty uses the provider's default endpoint."
                            >
                                <Input placeholder="https://api.example.com" autoComplete="off" />
                            </Form.Item>
                            <ProviderKeyField provider={provider} />
                        </div>
                        <SaveFooter status={status} saving={saving} hasError={hasError} className="mt-3" />
                    </div>
                    <Divider />
                </>
            )}
            <Flex align="center" justify="space-between" className="mb-3">
                <SectionHeading icon={ApiOutlined}>Models</SectionHeading>
                <Button size="small" icon={<ApiOutlined aria-hidden="true" />} onClick={onAdd}>
                    Add model
                </Button>
            </Flex>
            {models.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No models for this provider yet" />
            ) : (
                <Table<LLMModel> rowKey="id" size="small" columns={columns} dataSource={models} pagination={false} />
            )}
        </div>
    )
}
