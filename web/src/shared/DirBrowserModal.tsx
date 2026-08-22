import { useCallback, useEffect, useState } from 'react'
import { Modal } from 'antd'
import { api } from '../api/client'
import { DirList } from './DirList'

// DirBrowserModal is the directory picker behind WikiPathInput's folder icon:
// it lists a directory's subdirectories via GET /api/fs/dirs (DirList), walks
// up with the Up button, and reports the chosen directory through onSelect.
// The starting directory (initial, home being the fallback) is loaded on open.
export function DirBrowserModal({
    open,
    initial,
    onCancel,
    onSelect
}: {
    open: boolean
    initial: string
    onCancel: () => void
    onSelect: (path: string) => void
}) {
    const [current, setCurrent] = useState('')
    const [dirs, setDirs] = useState<string[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    const load = useCallback(async (path: string) => {
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
    }, [])

    // The browser starts at the current value; the home directory is the
    // fallback for an empty field.
    useEffect(() => {
        if (open) void load(initial !== '' ? initial : '~')
    }, [open, initial, load])

    const up = () => {
        const parent = current.split('/').slice(0, -1).join('/') || '/'
        void load(parent)
    }

    return (
        <Modal
            title="Select directory"
            open={open}
            onCancel={onCancel}
            onOk={() => onSelect(current)}
            okText="Select this directory"
            okButtonProps={{ disabled: current === '' }}
            destroyOnHidden
        >
            <DirList
                current={current}
                dirs={dirs}
                loading={loading}
                error={error}
                onPick={(d) => void load(d)}
                onUp={up}
            />
        </Modal>
    )
}
