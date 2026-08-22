import { useState } from 'react'
import { Input } from 'antd'
import { FolderOpenOutlined } from '@ant-design/icons'
import { DirBrowserModal } from './DirBrowserModal'

// WikiPathInput is the wiki path field with a clickable folder prefix icon
// that opens the DirBrowserModal directory picker. The value stays
// hand-editable at all times — the picker only fills it.
export function WikiPathInput({ value, onChange }: { value?: string; onChange?: (v: string) => void }) {
    const [open, setOpen] = useState(false)

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
                        onClick={() => setOpen(true)}
                    />
                }
            />
            <DirBrowserModal
                open={open}
                initial={value ?? ''}
                onCancel={() => setOpen(false)}
                onSelect={(path) => {
                    onChange?.(path)
                    setOpen(false)
                }}
            />
        </>
    )
}
