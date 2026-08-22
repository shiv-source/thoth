import { useEffect, useMemo, useState } from 'react'
import { Card, Divider, Flex, Form, Select } from 'antd'
import { BookOutlined, RocketOutlined, SettingOutlined } from '@ant-design/icons'
import { fetchModels, selectHealth, selectModelGroups, selectSettings } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { WikiPathInput } from '../../shared/WikiPathInput'
import { CardTitle } from './components/CardTitle'
import { SaveFooter } from './components/SaveFooter'
import { SectionHeading } from './components/SectionHeading'
import { SettingsShell } from './SettingsShell'
import { useSettingsForm } from './useSettingsForm'

// The scaffold folder set offered as suggestions in the General tab; the
// server keeps these defaults whenever wiki_folders is unset.
const defaultFolders = ['inbox', 'meetings', 'projects', 'links', 'setup', 'knowledge', 'todos', 'daily']

// SettingsGeneralPage is the wiki path + provider/model pickers in one card,
// with the save button and the saved/error feedback under them. Credentials
// and the model registry live on the Providers page.
export function SettingsGeneralPage() {
    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)
    const groups = useAppSelector(selectModelGroups)
    const { form, status, saving, hasError, save } = useSettingsForm()
    // The server reports the wiki default for the mode it runs in; the
    // fallback covers the render before the first health response lands.
    const defaultWiki = useAppSelector(selectHealth)?.default_wiki_path ?? '~/.thoth/wiki'

    useEffect(() => {
        void dispatch(fetchModels())
    }, [dispatch])

    // The model picker is a two-field cascade: the Provider select is view
    // state (which provider's models the Model select shows), the Model
    // select is the form field that gets saved.
    const [provider, setProvider] = useState<string | null>(null)

    // Resolve the saved model's provider once settings arrive, so an already
    // selected model stays visible in the cascade.
    useEffect(() => {
        if (!settings.data || settings.data.model === '') return
        const g = groups.find((gr) => gr.models.some((m) => m.value === settings.data!.model))
        if (g) setProvider(g.provider)
    }, [settings.data, groups])

    // Provider options are the registry's provider labels (A→Z, as the
    // server groups them); the current provider's models become the Model
    // select's options, with name + tag so optionRender can show the tag.
    const providerOptions = useMemo(
        () => groups.filter((g) => g.provider !== '').map((g) => ({ label: g.provider, value: g.provider })),
        [groups]
    )
    const modelOptions = useMemo(() => {
        const g = groups.find((gr) => gr.provider === provider)
        return (g?.models ?? []).map((m) => ({ label: m.name, value: m.value, tag: m.tag }))
    }, [groups, provider])

    const onProviderChange = (value: string) => {
        setProvider(value)
        // A model from the previous provider no longer applies; clear it so
        // the user picks one from the newly selected provider.
        form.setFieldValue('model', undefined)
    }

    return (
        <SettingsShell active="general">
            <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
                <Card size="small" title={<CardTitle icon={SettingOutlined}>General</CardTitle>}>
                    <SectionHeading icon={BookOutlined}>Knowledge base</SectionHeading>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Form.Item
                            label="Wiki path"
                            name="wiki_path"
                            extra={`Where your notes live on disk. Defaults to ${defaultWiki}.`}
                        >
                            <WikiPathInput />
                        </Form.Item>
                    </div>
                    <Form.Item
                        label="Scaffold folders"
                        name="wiki_folders"
                        extra="The folders created when a wiki is scaffolded. Type your own set or keep the defaults."
                    >
                        <Select
                            virtual={false}
                            mode="tags"
                            placeholder="inbox, meetings, projects, …"
                            options={defaultFolders.map((f) => ({ value: f }))}
                            tokenSeparators={[',']}
                        />
                    </Form.Item>
                    <Divider />
                    <SectionHeading icon={RocketOutlined}>Default model</SectionHeading>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Form.Item
                            label="Provider"
                            htmlFor="settings-provider"
                            extra="The provider your selected model comes from."
                        >
                            <Select
                                id="settings-provider"
                                virtual={false}
                                placeholder="Select a provider"
                                options={providerOptions}
                                notFoundContent="No models — add a provider model in the Providers tab"
                                value={provider ?? undefined}
                                onChange={onProviderChange}
                            />
                        </Form.Item>
                        <Form.Item label="Model" name="model" extra="Applied to all chats after the app restarts.">
                            <Select
                                virtual={false}
                                options={modelOptions}
                                notFoundContent="Select a provider first"
                                optionRender={(option) => (
                                    <Flex align="center" gap={8}>
                                        <span className="min-w-0 flex-1 truncate">{option.label}</span>
                                        {typeof option.data === 'object' &&
                                            option.data !== null &&
                                            (option.data as { tag?: string }).tag !== '' && (
                                                <span className="shrink-0 text-xs text-subtle">
                                                    {(option.data as { tag?: string }).tag}
                                                </span>
                                            )}
                                    </Flex>
                                )}
                            />
                        </Form.Item>
                    </div>
                    <Divider />
                    <SaveFooter status={status} saving={saving} hasError={hasError} />
                </Card>
            </Form>
        </SettingsShell>
    )
}
