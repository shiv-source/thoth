import { useMemo } from 'react'
import { Alert, AutoComplete, Form } from 'antd'
import { GlobalOutlined, LockOutlined } from '@ant-design/icons'
import type { GitHubRepo } from '../../../api/client'

// GitRemoteSection holds the sync repo URL input; suggestions come from the
// connected account, and a picked public repo triggers the security warning.
export function GitRemoteSection({
    repos,
    selectedRepo,
    onSelect,
    publicSelected
}: {
    repos: GitHubRepo[] | null
    selectedRepo: GitHubRepo | null
    onSelect: (repo: GitHubRepo | null) => void
    publicSelected: boolean
}) {
    const repoOptions = useMemo(
        () =>
            (repos ?? []).map((r) => ({
                value: r.clone_url,
                repo: r,
                label: (
                    <span className="flex items-start gap-2">
                        {r.private ? (
                            <LockOutlined aria-hidden="true" className="mt-0.5 shrink-0 text-subtle" />
                        ) : (
                            <GlobalOutlined aria-hidden="true" className="mt-0.5 shrink-0 text-subtle" />
                        )}
                        <span className="min-w-0 flex-1">
                            <span className="block truncate text-sm">{r.full_name}</span>
                            {r.description !== '' && (
                                <span className="block truncate text-xs text-subtle">{r.description}</span>
                            )}
                        </span>
                    </span>
                )
            })),
        [repos]
    )

    return (
        <>
            <Form.Item label="Git remote URL" name="repo_url">
                <AutoComplete
                    virtual={false}
                    placeholder="https://github.com/you/wiki.git"
                    options={repoOptions}
                    onSelect={(_value, option) => {
                        const repo = (option as { repo?: GitHubRepo }).repo
                        onSelect(repo ?? null)
                    }}
                    onChange={(value) => {
                        if (selectedRepo && selectedRepo.clone_url !== value) onSelect(null)
                    }}
                />
            </Form.Item>
            {publicSelected && (
                <Alert
                    type="error"
                    showIcon
                    title="Syncing to a public repository is blocked for your security — use a private repository."
                />
            )}
        </>
    )
}
