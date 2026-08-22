import { Tooltip } from 'antd'
import { FileTextOutlined, WarningOutlined } from '@ant-design/icons'

// TreeNodeLabel renders one tree row's title: a file icon for leaves, a
// warning Tooltip for unreadable directories (so the rest of the tree still
// renders — see internal/wiki tree()), and a file-count Tooltip for readable
// directories. The single icon per row convention lives here.
export function TreeNodeLabel({
    title,
    isLeaf,
    error,
    fileCount
}: {
    title: string
    isLeaf: boolean
    error?: string
    fileCount?: number
}) {
    if (isLeaf) {
        return (
            <span className="inline-flex items-center gap-1.5">
                <FileTextOutlined aria-hidden="true" className="text-subtle" />
                <span>{title}</span>
            </span>
        )
    }
    if (error) {
        return (
            <Tooltip title={error}>
                <span className="inline-flex items-center gap-1.5">
                    <WarningOutlined aria-hidden="true" className="text-amber-500" />
                    <span>{title}</span>
                </span>
            </Tooltip>
        )
    }
    const count = fileCount ?? 0
    return <Tooltip title={`${count} file${count === 1 ? '' : 's'}`}>{title}</Tooltip>
}
