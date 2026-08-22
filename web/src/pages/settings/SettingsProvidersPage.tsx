import { useEffect, useMemo, useState } from 'react'
import type { TableProps } from 'antd'
import { App, Button, Card, Collapse, Empty, Flex, Form, Popconfirm, Tag, Typography } from 'antd'
import { ApiOutlined } from '@ant-design/icons'
import type { LLMModel, ModelInput } from '../../api/client'
import {
    createModel,
    deleteModel,
    fetchModels,
    fetchSettings,
    selectModelGroups,
    selectModelList,
    selectSettings,
    updateModel
} from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CardTitle } from './components/CardTitle'
import { ModelModal } from './components/ModelModal'
import { ProviderHeader } from './components/ProviderHeader'
import { ProviderPanel } from './components/ProviderPanel'
import { tagColor } from './components/tagColor'
import { SettingsShell } from './SettingsShell'
import { useSettingsForm } from './useSettingsForm'

// SettingsProvidersPage is the provider management surface: a fallback API
// key for providers without a key of their own, then one collapsible panel
// per provider holding its credential form (base URL + API key) and the
// models registered under it. Every model mutation refetches the registry
// (the server re-groups and re-sorts) and settings (a rename may have moved
// the selected-model setting, a delete may have cleared it).
export function SettingsProvidersPage() {
    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)
    const models = useAppSelector(selectModelList)
    const groups = useAppSelector(selectModelGroups)
    const { form, status, saving, hasError, save } = useSettingsForm()
    const { message } = App.useApp()
    const [open, setOpen] = useState(false)
    const [editing, setEditing] = useState<LLMModel | null>(null)
    const [modelForm] = Form.useForm<ModelInput>()

    useEffect(() => {
        void dispatch(fetchModels())
    }, [dispatch])

    // Panels are the union of registry providers (the model groups, already
    // A→Z) and any leftover settings keys, so a provider keeps its panel even
    // when its last model is deleted.
    const providerNames = useMemo(() => {
        const names = groups.map((g) => g.provider)
        for (const p of Object.keys(settings.data?.providers ?? {})) {
            if (!names.includes(p)) names.push(p)
        }
        return names
    }, [groups, settings.data])

    // The tag dropdown offers every tag already in the registry (sorted);
    // AutoComplete also lets the user type a new one.
    const tagOptions = useMemo(
        () =>
            [...new Set(groups.flatMap((g) => g.models.map((m) => m.tag)).filter((t) => t !== ''))]
                .sort()
                .map((t) => ({ value: t })),
        [groups]
    )

    const openAdd = (provider: string) => {
        setEditing(null)
        modelForm.resetFields()
        modelForm.setFieldsValue({ provider })
        setOpen(true)
    }

    const openEdit = (m: LLMModel) => {
        setEditing(m)
        modelForm.setFieldsValue(m)
        setOpen(true)
    }

    const submit = async () => {
        const values = await modelForm.validateFields()
        try {
            if (editing) {
                await dispatch(updateModel({ id: editing.id, input: values })).unwrap()
            } else {
                await dispatch(createModel(values)).unwrap()
            }
            void dispatch(fetchModels())
            void dispatch(fetchSettings())
            setOpen(false)
            void message.success(editing ? 'Model updated' : 'Model added')
        } catch {
            void message.error('Could not save model')
        }
    }

    const remove = async (m: LLMModel) => {
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
                    <Button size="small" onClick={() => openEdit(m)}>
                        Edit
                    </Button>
                    <Popconfirm title="Delete this model?" trigger="click" onConfirm={() => void remove(m)}>
                        <Button size="small" danger>
                            Delete
                        </Button>
                    </Popconfirm>
                </Flex>
            )
        }
    ]

    return (
        <SettingsShell active="providers">
            <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
                <Card size="small" title={<CardTitle icon={ApiOutlined}>Providers</CardTitle>}>
                    {providerNames.length === 0 ? (
                        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No models yet">
                            <Button type="primary" onClick={() => openAdd('')}>
                                Add your first model
                            </Button>
                        </Empty>
                    ) : (
                        <Collapse
                            defaultActiveKey={[providerNames[0]!]}
                            items={providerNames.map((provider) => ({
                                key: provider,
                                label: (
                                    <ProviderHeader
                                        provider={provider}
                                        modelCount={models.filter((m) => m.provider === provider).length}
                                        hasKey={settings.data?.providers?.[provider]?.has_api_key === true}
                                        baseURL={settings.data?.providers?.[provider]?.base_url ?? ''}
                                    />
                                ),
                                children: (
                                    <ProviderPanel
                                        provider={provider}
                                        models={models.filter((m) => m.provider === provider)}
                                        columns={columns}
                                        status={status}
                                        saving={saving}
                                        hasError={hasError}
                                        onAdd={() => openAdd(provider)}
                                    />
                                )
                            }))}
                        />
                    )}
                    <ModelModal
                        open={open}
                        editing={editing}
                        form={modelForm}
                        tagOptions={tagOptions}
                        onCancel={() => setOpen(false)}
                        onOk={() => void submit()}
                    />
                </Card>
            </Form>
        </SettingsShell>
    )
}
