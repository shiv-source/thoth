import { Form, Switch } from 'antd'

// GitSyncSection is the single auto-sync switch.
export function GitSyncSection() {
    return (
        <Form.Item
            name="sync_enabled"
            valuePropName="checked"
            label="Auto-sync the wiki to the remote"
            extra="Stores your wiki in a remote git repository. Thoth initializes the repo if needed, commits the current tree, and pushes the branch."
        >
            <Switch />
        </Form.Item>
    )
}
