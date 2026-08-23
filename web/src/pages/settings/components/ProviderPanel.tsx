import type { TableProps } from 'antd'
import { Button, Empty, Flex, Table } from 'antd'
import { ApiOutlined } from '@ant-design/icons'
import type { LLMModel } from '../../../api/client'
import { SectionHeading } from './SectionHeading'

// ProviderPanel is one Collapse body: the provider's registered models table
// with its add/edit/delete actions. Credentials live on the provider row and
// are edited through the provider modal, not here.
export function ProviderPanel({
    models,
    columns,
    onAdd
}: {
    models: LLMModel[]
    columns: TableProps<LLMModel>['columns']
    onAdd: () => void
}) {
    return (
        <div className="grid gap-4">
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
