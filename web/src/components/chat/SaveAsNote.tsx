import { useState } from 'react'
import { App, Button, Modal, Select, Tooltip } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { api } from '../../api/client'
import { fetchSettings, fetchTree, notify, selectSettings } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { DEFAULT_WIKI_FOLDERS } from '../../shared/wikiFolders'

// SaveAsNote promotes an assistant message into a permanent wiki note: the
// user picks the target folder (defaulting to the first configured folder),
// and the server files it via the same save path the assistant's own saves
// use (frontmatter, folder-appropriate type, kebab filename). The saved path
// is toasted via the notifications store and the tree refetches so the note
// shows up immediately.
export function SaveAsNote({ content }: { content: string }) {
    const { message } = App.useApp()
    const dispatch = useAppDispatch()
    const settings = useAppSelector(selectSettings)
    const [open, setOpen] = useState(false)
    const [folder, setFolder] = useState<string | null>(null)
    const [saving, setSaving] = useState(false)

    // The picker lists the configured folders; before settings arrive it
    // falls back to the scaffold defaults (which the server keeps when
    // wiki_folders is unset). Settings are refetched on open so a folder
    // configured elsewhere shows up here too. The default folder is derived
    // live (first configured) rather than pinned, so it follows settings
    // arriving after the modal opens.
    const configured = settings.data?.wiki_folders?.length ? settings.data.wiki_folders : DEFAULT_WIKI_FOLDERS
    const folders = configured.filter((f) => f !== 'attachments')
    const chosen = folder ?? folders[0] ?? 'inbox'

    const openModal = () => {
        setOpen(true)
        setFolder(null) // re-default from settings on each open
        if (!settings.data) void dispatch(fetchSettings())
    }

    const save = async () => {
        setSaving(true)
        try {
            const res = await api.saveNote({ content, folder: chosen })
            dispatch(notify({ kind: 'note', title: 'Note saved', body: res.path }))
            void dispatch(fetchTree())
            setOpen(false)
        } catch {
            void message.error('Could not save the note')
        } finally {
            setSaving(false)
        }
    }

    return (
        <>
            <Tooltip title="Save as note">
                <span className="absolute right-8 top-2">
                    <Button
                        type="text"
                        size="small"
                        aria-label="Save as note"
                        icon={<SaveOutlined aria-hidden="true" />}
                        onClick={openModal}
                    />
                </span>
            </Tooltip>
            <Modal
                title="Save as note"
                open={open}
                onOk={() => void save()}
                onCancel={() => setOpen(false)}
                okText="Save"
                confirmLoading={saving}
                destroyOnHidden
            >
                <div className="mt-4 flex flex-col gap-2">
                    <span className="text-sm text-subtle">Target folder</span>
                    <Select
                        virtual={false}
                        aria-label="Target folder"
                        value={chosen}
                        onChange={setFolder}
                        options={folders.map((f) => ({ value: f, label: f }))}
                        placeholder="Choose a folder"
                    />
                </div>
            </Modal>
        </>
    )
}
