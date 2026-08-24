import { useEffect, useMemo, useState } from 'react'
import { App, Button, Card, Divider, Flex, Form, Progress, Select, Switch, Upload } from 'antd'
import { BookOutlined, DownloadOutlined, RobotOutlined, SettingOutlined, UploadOutlined } from '@ant-design/icons'
import { api } from '../../api/client'
import { fetchModels, selectHealth, selectModelGroups, selectSettings } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { WikiPathInput } from '../../shared/WikiPathInput'
import { DEFAULT_WIKI_FOLDERS } from '../../shared/wikiFolders'
import { CardTitle } from './components/CardTitle'
import { FormSection } from './components/FormSection'
import { SaveFooter } from './components/SaveFooter'
import { SettingsShell } from './SettingsShell'
import { useSettingsForm } from './useSettingsForm'

// SettingsGeneralPage is the wiki path + model configuration in one card,
// grouped into enterprise sections (Knowledge base / Default model & context
// / Backup & transfer) with a shared save bar at the bottom. Credentials and
// the model registry live on the Providers page.
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

    const { message } = App.useApp()
    const [exporting, setExporting] = useState(false)
    const [importing, setImporting] = useState(false)
    const [importProgress, setImportProgress] = useState(0)

    // export downloads the wiki as a zip; history pulls in .git so git
    // history travels with it.
    const exportWiki = async (history: boolean) => {
        setExporting(true)
        try {
            await api.exportWiki(history)
            void message.success('Wiki exported')
        } catch {
            void message.error('Could not export the wiki')
        } finally {
            setExporting(false)
        }
    }

    // doImport uploads a selected zip. beforeUpload (below) already stopped
    // antd's auto-upload, so this runs the request and reports progress.
    const doImport = async (file: File) => {
        setImporting(true)
        setImportProgress(0)
        try {
            const res = await api.importWiki(file, setImportProgress)
            void message.success(`Imported ${res.files} files${res.backup ? ' — a backup was saved' : ''}`)
        } catch (e) {
            void message.error(e instanceof Error ? e.message : 'Could not import the wiki')
        } finally {
            setImporting(false)
        }
    }

    return (
        <SettingsShell active="general">
            <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
                <Card title={<CardTitle icon={SettingOutlined}>General</CardTitle>}>
                    <FormSection
                        icon={BookOutlined}
                        title="Knowledge base"
                        description="Where your notes live on disk and the folders a new wiki is scaffolded with."
                    >
                        <div className="rounded-lg border border-line bg-raised p-5">
                            <div className="grid gap-4 md:grid-cols-2">
                                <Form.Item
                                    label="Wiki path"
                                    name="wiki_path"
                                    extra={`Defaults to ${defaultWiki}.`}
                                    className="min-w-0"
                                >
                                    <WikiPathInput />
                                </Form.Item>
                                <Form.Item
                                    label="Scaffold folders"
                                    name="wiki_folders"
                                    extra="Type your own set or keep the defaults."
                                    className="min-w-0"
                                >
                                    <Select
                                        virtual={false}
                                        mode="tags"
                                        placeholder="inbox, meetings, projects, …"
                                        options={DEFAULT_WIKI_FOLDERS.map((f) => ({ value: f }))}
                                        tokenSeparators={[',']}
                                    />
                                </Form.Item>
                            </div>
                        </div>
                    </FormSection>

                    <Divider className="my-8" />

                    <FormSection
                        icon={RobotOutlined}
                        title="Default model & context"
                        description="The model used for new chats and how the assistant prepares each turn."
                    >
                        <div className="rounded-lg border border-line bg-raised p-5">
                            <div className="grid gap-4 md:grid-cols-2">
                                <Form.Item
                                    label="Provider"
                                    htmlFor="settings-provider"
                                    extra="The provider your selected model comes from."
                                    className="min-w-0"
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
                                <Form.Item
                                    label="Model"
                                    name="model"
                                    extra="Applied to all chats after the app restarts."
                                    className="min-w-0"
                                >
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
                            <Form.Item
                                label="Context injection"
                                name="context_injection"
                                valuePropName="checked"
                                extra="Pre-search the wiki into each turn so answers start faster. Off by default — it changes how the assistant answers."
                                className="max-w-lg"
                            >
                                <Switch />
                            </Form.Item>
                        </div>
                    </FormSection>

                    <Divider className="my-8" />

                    <FormSection
                        icon={UploadOutlined}
                        title="Backup & transfer"
                        description="Export downloads the wiki as a zip; Import merges a zip back in, backing up the current wiki first."
                    >
                        <div className="rounded-lg border border-line bg-raised p-5">
                            <div className="max-w-lg">
                                <Flex gap={8} wrap="wrap">
                                    <Button
                                        icon={<DownloadOutlined aria-hidden="true" />}
                                        loading={exporting}
                                        onClick={() => void exportWiki(false)}
                                    >
                                        Export
                                    </Button>
                                    <Button
                                        icon={<DownloadOutlined aria-hidden="true" />}
                                        loading={exporting}
                                        onClick={() => void exportWiki(true)}
                                    >
                                        Export with history
                                    </Button>
                                    <Upload
                                        accept=".zip,application/zip"
                                        showUploadList={false}
                                        beforeUpload={(file) => {
                                            void doImport(file)
                                            return false
                                        }}
                                    >
                                        <Button
                                            icon={<UploadOutlined aria-hidden="true" />}
                                            loading={importing}
                                            disabled={exporting}
                                        >
                                            Import
                                        </Button>
                                    </Upload>
                                </Flex>
                                {importing && (
                                    <Progress className="mt-3" percent={importProgress} status="active" size="small" />
                                )}
                            </div>
                        </div>
                    </FormSection>

                    <Divider className="my-8" />
                    <SaveFooter status={status} saving={saving} hasError={hasError} />
                </Card>
            </Form>
        </SettingsShell>
    )
}
