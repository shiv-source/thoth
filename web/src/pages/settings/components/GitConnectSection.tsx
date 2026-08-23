import { Alert, Button, Card, Form, Input } from 'antd'
import { GithubOutlined } from '@ant-design/icons'
import { CardTitle } from './CardTitle'
import { SectionHeading } from './SectionHeading'

// GitConnectSection is the not-yet-connected Git page card: the personal
// access token input (local state owned by the page) with the Connect
// button. The token is stored server-side in the github_auth row.
export function GitConnectSection({
    token,
    onTokenChange,
    connecting,
    error,
    onConnect
}: {
    token: string
    onTokenChange: (value: string) => void
    connecting: boolean
    error: string | null
    onConnect: () => void
}) {
    return (
        <Card size="small" title={<CardTitle icon={GithubOutlined}>GitHub</CardTitle>}>
            <SectionHeading icon={GithubOutlined}>Account</SectionHeading>
            <Form.Item
                label="Personal access token"
                extra={
                    <>
                        Connect your GitHub account to store the sync repo URL and credentials. The token needs the{' '}
                        <code>user:email</code> scope and is stored locally in thoth.db — it is never sent anywhere
                        except api.github.com.
                    </>
                }
            >
                <Input.Password placeholder="ghp_…" value={token} onChange={(e) => onTokenChange(e.target.value)} />
            </Form.Item>
            {error && <Alert type="error" showIcon title={error} className="mb-4" />}
            <div className="flex justify-end">
                <Button type="primary" loading={connecting} onClick={onConnect}>
                    Connect
                </Button>
            </div>
        </Card>
    )
}
