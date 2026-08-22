import { useEffect, useState } from 'react'
import { Alert, App, Button, Card, Divider, Form } from 'antd'
import { BranchesOutlined, GithubOutlined, SyncOutlined } from '@ant-design/icons'
import type { GitHubRepo } from '../../api/client'
import {
    connectGit,
    disconnectGit,
    fetchGitAuth,
    fetchGitRepos,
    pushWiki,
    selectGitAuth,
    selectGitConnecting,
    selectGitError,
    selectGitPushing,
    selectGitRepos
} from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { CardTitle } from './components/CardTitle'
import { GitAccountSection } from './components/GitAccountSection'
import { GitConnectSection } from './components/GitConnectSection'
import { GitRemoteSection } from './components/GitRemoteSection'
import { GitSyncSection } from './components/GitSyncSection'
import { SectionHeading } from './components/SectionHeading'
import { SettingsShell } from './SettingsShell'
import { useSettingsForm } from './useSettingsForm'

// SettingsGitPage connects a GitHub account (the token is stored
// server-side), keeps the sync repo URL in the settings form (Save persists
// it to the DB), and "Initialize & Push" runs the server-side setup against
// the current wiki path. The URL input only appears once connected: the
// server stores it in the github_auth row.
export function SettingsGitPage() {
    const dispatch = useAppDispatch()
    const { message } = App.useApp()
    const { form, save } = useSettingsForm()
    const auth = useAppSelector(selectGitAuth)
    const repos = useAppSelector(selectGitRepos)
    const connecting = useAppSelector(selectGitConnecting)
    const pushing = useAppSelector(selectGitPushing)
    const gitError = useAppSelector(selectGitError)
    const [token, setToken] = useState('')
    // Only a repo picked from the suggestions is classified: a hand-typed
    // URL gets no visibility warning (it cannot be verified here).
    const [selectedRepo, setSelectedRepo] = useState<GitHubRepo | null>(null)

    const connected = auth !== null && auth.username !== ''

    useEffect(() => {
        void dispatch(fetchGitAuth())
    }, [dispatch])

    // Suggestions for the repo URL come from the connected account; a failed
    // load just leaves the list empty — typing a URL always works.
    useEffect(() => {
        if (connected) void dispatch(fetchGitRepos())
    }, [connected, dispatch])

    const publicSelected = selectedRepo !== null && !selectedRepo.private

    const connect = async () => {
        if (!token) {
            void message.error('Enter a personal access token.')
            return
        }
        try {
            await dispatch(connectGit(token)).unwrap()
            setToken('')
            void message.success('GitHub connected')
        } catch (e) {
            void message.error(e instanceof Error ? e.message : 'Could not connect GitHub')
        }
    }

    const disconnect = async () => {
        try {
            await dispatch(disconnectGit()).unwrap()
            form.setFieldsValue({ repo_url: '' })
            void message.success('GitHub disconnected')
        } catch {
            void message.error('Could not disconnect GitHub')
        }
    }

    const push = async () => {
        const url = form.getFieldValue('repo_url') as string
        if (!url) {
            void message.error('Enter a remote URL first.')
            return
        }
        try {
            const res = await dispatch(pushWiki(url)).unwrap()
            if (res.ok) {
                void message.success('Wiki pushed to remote')
            } else {
                void message.error(res.error ?? 'Could not push the wiki')
            }
        } catch {
            void message.error('Could not push the wiki')
        }
    }

    if (!connected) {
        return (
            <SettingsShell active="git">
                <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
                    <GitConnectSection
                        token={token}
                        onTokenChange={setToken}
                        connecting={connecting}
                        error={gitError}
                        onConnect={() => void connect()}
                    />
                </Form>
            </SettingsShell>
        )
    }

    return (
        <SettingsShell active="git">
            <Form form={form} layout="vertical" onFinish={(values) => void save(values)}>
                <Card size="small" title={<CardTitle icon={GithubOutlined}>GitHub</CardTitle>}>
                    <SectionHeading icon={GithubOutlined}>Account</SectionHeading>
                    <GitAccountSection auth={auth} onDisconnect={() => void disconnect()} />
                    <Divider />
                    <div className="grid gap-4 md:grid-cols-2">
                        <div>
                            <SectionHeading icon={BranchesOutlined}>Remote</SectionHeading>
                            <GitRemoteSection
                                repos={repos}
                                selectedRepo={selectedRepo}
                                onSelect={setSelectedRepo}
                                publicSelected={publicSelected}
                            />
                        </div>
                        <div>
                            <SectionHeading icon={SyncOutlined}>Sync</SectionHeading>
                            <GitSyncSection />
                        </div>
                    </div>
                    <Divider />
                    {gitError && <Alert type="error" showIcon message={gitError} />}
                    <div className="flex justify-end gap-3">
                        <Button htmlType="submit">Save</Button>
                        <Button type="primary" loading={pushing} disabled={publicSelected} onClick={() => void push()}>
                            Initialize & Push
                        </Button>
                    </div>
                </Card>
            </Form>
        </SettingsShell>
    )
}
