import { Avatar, Button, Flex, Tag } from 'antd'
import { ClockCircleOutlined, GithubOutlined } from '@ant-design/icons'
import type { GitHubIdentity } from '../../../api/client'

// GitAccountSection shows the connected identity (avatar, profile link,
// account dates, scopes) with the disconnect action.
export function GitAccountSection({ auth, onDisconnect }: { auth: GitHubIdentity; onDisconnect: () => void }) {
    const scopes = auth.scopes
        .split(',')
        .map((s) => s.trim())
        .filter((s) => s !== '')

    return (
        <div className="flex items-center gap-4 rounded-md border border-line bg-raised p-4">
            {auth.avatar_url !== '' ? (
                <Avatar src={auth.avatar_url} size={48} />
            ) : (
                <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent">
                    <GithubOutlined aria-hidden="true" className="text-xl" />
                </div>
            )}
            <div className="min-w-0 flex-1">
                <Flex align="center" gap={8}>
                    <p className="truncate text-base font-semibold text-heading">
                        {auth.profile_url !== '' ? (
                            <a href={auth.profile_url} target="_blank" rel="noreferrer" className="hover:underline">
                                {auth.display_name || auth.username}
                            </a>
                        ) : (
                            auth.display_name || auth.username
                        )}
                    </p>
                    <Tag color="success">Connected</Tag>
                </Flex>
                <p className="mt-0.5 truncate text-xs text-subtle">{auth.email || auth.username}</p>
                {(auth.account_created_at !== '' || auth.account_updated_at !== '') && (
                    <Flex align="center" gap={4} className="mt-1">
                        <ClockCircleOutlined aria-hidden="true" className="text-subtle" />
                        <p className="truncate text-xs text-subtle">
                            Member since {auth.account_created_at.slice(0, 10)}
                            {auth.account_updated_at !== '' && ` · Updated ${auth.account_updated_at.slice(0, 10)}`}
                        </p>
                    </Flex>
                )}
                {scopes.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                        {scopes.map((s) => (
                            <Tag key={s}>{s}</Tag>
                        ))}
                    </div>
                )}
            </div>
            <Button onClick={onDisconnect}>Disconnect</Button>
        </div>
    )
}
