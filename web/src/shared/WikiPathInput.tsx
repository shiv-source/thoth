import { useState } from 'react'
import { Alert, Button, Flex, Input, Modal, Typography } from 'antd'
import { ArrowUpOutlined, FolderOpenOutlined } from '@ant-design/icons'
import { api } from '../api/client'

// WikiPathInput is the wiki path field with a clickable folder prefix icon
// that opens a directory browser (the local server lists the filesystem via
// GET /api/fs/dirs). The value stays hand-editable at all times — the picker
// only fills it.
export function WikiPathInput({ value, onChange }: { value?: string; onChange?: (v: string) => void }) {
    const [open, setOpen] = useState(false)
    const [current, setCurrent] = useState('')
    const [dirs, setDirs] = useState<string[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const load = async (path: string) => {
        setLoading(true)
        setError(null)
        try {
            const { dirs: next } = await api.listDirs(path)
            setCurrent(path)
            setDirs(next)
        } catch {
            setError('Cannot read this directory.')
        } finally {
            setLoading(false)
        }
    }

    const openPicker = () => {
        setOpen(true)
        // The browser starts at the current value; the home directory is
        // the fallback for an empty field.
        void load(value && value !== '' ? value : '~')
    }

    const up = () => {
        const parent = current.split('/').slice(0, -1).join('/') || '/'
        void load(parent)
    }

    return (
        <>
            <Input
                value={value}
                onChange={(e) => onChange?.(e.target.value)}
                placeholder="~/.thoth/wiki"
                prefix={
                    <FolderOpenOutlined
                        aria-label="Choose wiki directory"
                        className="cursor-pointer text-subtle"
                        onClick={openPicker}
                    />
                }
            />
            <Modal
                title="Select directory"
                open={open}
                onCancel={() => setOpen(false)}
                onOk={() => {
                    onChange?.(current)
                    setOpen(false)
                }}
                okText="Select this directory"
                okButtonProps={{ disabled: current === '' }}
                destroyOnHidden
            >
                <Flex align="center" gap={8} className="mb-2">
                    <Typography.Text code ellipsis className="min-w-0 flex-1">
                        {current}
                    </Typography.Text>
                    <Button size="small" icon={<ArrowUpOutlined aria-hidden="true" />} onClick={() => void up()}>
                        Up
                    </Button>
                </Flex>
                {error !== null && <Alert type="error" showIcon message={error} className="mb-2" />}
                <div className="max-h-72 overflow-y-auto">
                    {dirs.map((d) => {
                        const name = d.split('/').pop() ?? d
                        return (
                            <Button
                                key={d}
                                type="text"
                                block
                                className="justify-start"
                                icon={<FolderOpenOutlined aria-hidden="true" className="text-subtle" />}
                                onClick={() => void load(d)}
                            >
                                <span className="truncate">{name}</span>
                            </Button>
                        )
                    })}
                    {dirs.length === 0 && !loading && error === null && (
                        <Typography.Text type="secondary">No subdirectories.</Typography.Text>
                    )}
                </div>
            </Modal>
        </>
    )
}
