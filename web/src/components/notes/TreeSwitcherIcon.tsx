import { FolderOpenOutlined, FolderOutlined } from '@ant-design/icons'

// TreeSwitcherIcon is the DirectoryTree expand/collapse caret replacement: an
// open or closed folder instead of antd's default arrow. showIcon is off on
// the tree, so this is the only icon per row.
export function TreeSwitcherIcon({ expanded }: { expanded?: boolean }) {
    return expanded ? (
        <FolderOpenOutlined aria-hidden="true" className="text-subtle" />
    ) : (
        <FolderOutlined aria-hidden="true" className="text-subtle" />
    )
}
