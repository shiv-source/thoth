import { Alert, Button, Typography } from 'antd'
import { ArrowUpOutlined, FolderOpenOutlined } from '@ant-design/icons'

// DirList is the directory picker body: the current path with the Up button,
// then the subdirectory rows (or a read error / "no subdirectories" state).
export function DirList({
    current,
    dirs,
    loading,
    error,
    onPick,
    onUp
}: {
    current: string
    dirs: string[]
    loading: boolean
    error: string | null
    onPick: (path: string) => void
    onUp: () => void
}) {
    return (
        <>
            <div className="mb-2 flex items-center gap-2">
                <Typography.Text code ellipsis className="min-w-0 flex-1">
                    {current}
                </Typography.Text>
                <Button size="small" icon={<ArrowUpOutlined aria-hidden="true" />} onClick={onUp}>
                    Up
                </Button>
            </div>
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
                            onClick={() => onPick(d)}
                        >
                            <span className="truncate">{name}</span>
                        </Button>
                    )
                })}
                {dirs.length === 0 && !loading && error === null && (
                    <Typography.Text type="secondary">No subdirectories.</Typography.Text>
                )}
            </div>
        </>
    )
}
