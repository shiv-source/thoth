import { describe, expect, it, vi } from 'vitest'
import { axiosError, axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import {
    connectGit,
    fetchGitAuth,
    fetchGitRepos,
    selectGitAuth,
    selectGitConnecting,
    selectGitError,
    selectGitRepos
} from '../index'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const identity = {
    username: 'shiv',
    display_name: 'Shiv',
    email: 'shiv@example.com',
    avatar_url: '',
    profile_url: '',
    scopes: 'repo',
    account_created_at: '',
    account_updated_at: ''
}

describe('gitSlice', () => {
    it('loads the connected identity', async () => {
        stubAPI(mocks, { '/api/github/auth': () => identity })
        const store = makeStore()
        await store.dispatch(fetchGitAuth())
        expect(selectGitAuth(store.getState())).toEqual(identity)
    })

    it('treats a missing identity as not-connected, not an error', async () => {
        stubAPI(mocks, {})
        const store = makeStore()
        await store.dispatch(fetchGitAuth()).catch(() => {})
        expect(selectGitAuth(store.getState())).toBeNull()
        expect(selectGitError(store.getState())).toBeNull()
    })

    it('loads the repo list', async () => {
        stubAPI(mocks, {
            '/api/github/repos': () => ({
                repos: [{ full_name: 'shiv/thoth', clone_url: 'x', private: false, description: '' }]
            })
        })
        const store = makeStore()
        await store.dispatch(fetchGitRepos())
        expect(selectGitRepos(store.getState())).toEqual([
            { full_name: 'shiv/thoth', clone_url: 'x', private: false, description: '' }
        ])
    })

    it('connects with a token and stores the identity', async () => {
        stubAPI(mocks, { 'POST /api/github/auth': () => identity })
        const store = makeStore()
        await store.dispatch(connectGit('token'))
        expect(selectGitConnecting(store.getState())).toBe(false)
        expect(selectGitAuth(store.getState())).toEqual(identity)
    })

    it('surfaces the server message when the connection fails', async () => {
        stubAPI(mocks, {})
        mocks.post.mockRejectedValueOnce(axiosError(401, { error: 'invalid token' }))
        const store = makeStore()
        await store.dispatch(connectGit('bad')).catch(() => {})
        expect(selectGitConnecting(store.getState())).toBe(false)
        expect(selectGitError(store.getState())).toBe('invalid token')
    })
})
