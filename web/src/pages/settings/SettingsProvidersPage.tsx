import { useEffect, useMemo, useState } from 'react'
import type { TableProps } from 'antd'
import { App, Button, Card, Collapse, Empty, Flex, Form, Popconfirm, Tag, Typography } from 'antd'
import { ApiOutlined, PlusOutlined } from '@ant-design/icons'
import type { LLMModel, ModelInput, Provider, ProviderInput } from '../../api/client'
import {
    createModel,
    createProvider,
    deleteModel,
    deleteProvider,
    fetchModels,
    fetchProviders,
    fetchSettings,
    selectModelGroups,
    selectModelList,
    selectProviders,
    updateModel,
    updateProvider
} from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CardTitle } from './components/CardTitle'
import { ModelModal } from './components/ModelModal'
import { ProviderHeader } from './components/ProviderHeader'
import { ProviderModal, type ProviderFormValues } from './components/ProviderModal'
import { ProviderPanel } from './components/ProviderPanel'
import { tagColor } from './components/tagColor'
import { SettingsShell } from './SettingsShell'

// SettingsProvidersPage is the provider management surface: a card of one
// collapsible panel per provider, each holding its models table. Providers
// are created first (a provider row exists before any model); credentials
// (base URL + API key) live on the provider row and are edited through the
// provider modal. Every mutation refetches the registry (the server
// re-groups and re-sorts) and, where a rename/delete may have moved the
// selected-model setting, the settings.
export function SettingsProvidersPage() {
    const dispatch = useAppDispatch()
    const models = useAppSelector(selectModelList)
    const groups = useAppSelector(selectModelGroups)
    const providers = useAppSelector(selectProviders)
    const { message } = App.useApp()

    const [providerOpen, setProviderOpen] = useState(false)
    const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
    const [providerForm] = Form.useForm<ProviderFormValues>()

    const [modelOpen, setModelOpen] = useState(false)
    const [editing, setEditing] = useState<LLMModel | null>(null)
    const [modelForm] = Form.useForm<ModelInput>()

    useEffect(() => {
        void dispatch(fetchModels())
        void dispatch(fetchProviders())
    }, [dispatch])

    // The tag dropdown offers every tag already in the registry (sorted);
    // AutoComplete also lets the user type a new one.
    const tagOptions = useMemo(
        () =>
            [...new Set(groups.flatMap((g) => g.models.map((m) => m.tag)).filter((t) => t !== ''))]
                .sort()
                .map((t) => ({ value: t })),
        [groups]
    )

    const openAddProvider = () => {
        setEditingProvider(null)
        providerForm.resetFields()
        setProviderOpen(true)
    }

    const openEditProvider = (p: Provider) => {
        setEditingProvider(p)
        providerForm.resetFields()
        providerForm.setFieldsValue({
            name: p.name,
            base_url: p.base_url,
            custom_headers: Object.entries(p.custom_headers).map(([name, value]) => ({ name, value }))
        })
        setProviderOpen(true)
    }

    const submitProvider = async () => {
        const values = await providerForm.validateFields()
        const input: ProviderInput = {
            name: values.name,
            base_url: values.base_url,
            api_key: values.api_key
        }
        // The key/value rows become the custom_headers object (empty rows are
        // dropped); an empty set clears the provider's headers on the server.
        const headers = Object.fromEntries(
            (values.custom_headers ?? []).filter((h) => h.name.trim() !== '').map((h) => [h.name.trim(), h.value])
        )
        if (Object.keys(headers).length > 0) input.custom_headers = headers
        try {
            if (editingProvider) {
                await dispatch(updateProvider({ id: editingProvider.id, input })).unwrap()
            } else {
                await dispatch(createProvider(input)).unwrap()
            }
            void dispatch(fetchProviders())
            void dispatch(fetchModels())
            setProviderOpen(false)
            void message.success(editingProvider ? 'Provider updated' : 'Provider added')
        } catch {
            void message.error('Could not save provider')
        }
    }

    const removeProvider = async (p: Provider) => {
        try {
            await dispatch(deleteProvider(p.id)).unwrap()
            void dispatch(fetchProviders())
            void dispatch(fetchModels())
            void dispatch(fetchSettings())
            void message.success('Provider deleted')
        } catch {
            void message.error('Could not delete provider')
        }
    }

    const openAddModel = (providerID: number) => {
        setEditing(null)
        modelForm.resetFields()
        modelForm.setFieldsValue({ provider_id: providerID })
        setModelOpen(true)
    }

    const openEditModel = (m: LLMModel) => {
        setEditing(m)
        modelForm.setFieldsValue(m)
        setModelOpen(true)
    }

    const submitModel = async () => {
        const values = await modelForm.validateFields()
        try {
            if (editing) {
                await dispatch(updateModel({ id: editing.id, input: values })).unwrap()
            } else {
                await dispatch(createModel(values)).unwrap()
            }
            void dispatch(fetchModels())
            void dispatch(fetchSettings())
            setModelOpen(false)
            void message.success(editing ? 'Model updated' : 'Model added')
        } catch {
            void message.error('Could not save model')
        }
    }

    const removeModel = async (m: LLMModel) => {
        try {
            await dispatch(deleteModel(m.id)).unwrap()
            void dispatch(fetchModels())
            void dispatch(fetchSettings())
            void message.success('Model deleted')
        } catch {
            void message.error('Could not delete model')
        }
    }

    // The per-provider table is the flat registry table without the (now
    // redundant) provider column; the panel header names the provider.
    const columns: TableProps<LLMModel>['columns'] = [
        { title: 'Name', dataIndex: 'name' },
        {
            title: 'Tag',
            dataIndex: 'tag',
            render: (value: string) =>
                value !== '' ? (
                    <Tag color={tagColor(value)}>{value}</Tag>
                ) : (
                    <Typography.Text type="secondary">—</Typography.Text>
                )
        },
        {
            title: 'Value',
            dataIndex: 'value',
            render: (value: string) => <code className="font-mono text-xs">{value}</code>
        },
        {
            title: '',
            key: 'actions',
            width: 150,
            render: (_, m) => (
                <Flex gap={4} justify="flex-end">
                    <Button size="small" onClick={() => openEditModel(m)}>
                        Edit
                    </Button>
                    <Popconfirm title="Delete this model?" trigger="click" onConfirm={() => void removeModel(m)}>
                        <Button size="small" danger>
                            Delete
                        </Button>
                    </Popconfirm>
                </Flex>
            )
        }
    ]

    // Models without a provider (provider_id 0 — the Unassigned catch-all)
    // keep a panel of their own so they never become invisible.
    const unassignedModels = models.filter((m) => m.provider_id === 0)
    const panelItems = [
        ...providers.map((p) => ({
            key: p.name,
            label: (
                <ProviderHeader
                    name={p.name}
                    modelCount={p.model_count}
                    hasKey={p.has_api_key}
                    baseURL={p.base_url}
                    hasCustomHeaders={Object.keys(p.custom_headers).length > 0}
                    onEdit={() => openEditProvider(p)}
                    onDelete={() => void removeProvider(p)}
                />
            ),
            children: (
                <ProviderPanel
                    models={models.filter((m) => m.provider_id === p.id)}
                    columns={columns}
                    onAdd={() => openAddModel(p.id)}
                />
            )
        })),
        ...(unassignedModels.length > 0
            ? [
                  {
                      key: '__unassigned__',
                      label: (
                          <ProviderHeader
                              name="Unassigned"
                              modelCount={unassignedModels.length}
                              hasKey={false}
                              baseURL=""
                          />
                      ),
                      children: (
                          <ProviderPanel models={unassignedModels} columns={columns} onAdd={() => openAddModel(0)} />
                      )
                  }
              ]
            : [])
    ]

    return (
        <SettingsShell active="providers">
            <Card
                title={<CardTitle icon={ApiOutlined}>Providers</CardTitle>}
                extra={
                    <Button type="primary" icon={<PlusOutlined aria-hidden="true" />} onClick={openAddProvider}>
                        Add provider
                    </Button>
                }
            >
                {panelItems.length === 0 ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No providers yet">
                        <Button type="primary" onClick={openAddProvider}>
                            Add your first provider
                        </Button>
                    </Empty>
                ) : (
                    <Collapse defaultActiveKey={[panelItems[0]!.key]} items={panelItems} />
                )}
                <ProviderModal
                    open={providerOpen}
                    editing={editingProvider}
                    form={providerForm}
                    onCancel={() => setProviderOpen(false)}
                    onOk={() => void submitProvider()}
                />
                <ModelModal
                    open={modelOpen}
                    editing={editing}
                    form={modelForm}
                    providers={providers}
                    tagOptions={tagOptions}
                    onCancel={() => setModelOpen(false)}
                    onOk={() => void submitModel()}
                />
            </Card>
        </SettingsShell>
    )
}
