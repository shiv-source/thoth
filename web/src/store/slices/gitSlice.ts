import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api, type GitHubIdentity, type GitHubRepo } from '../../api/client'
import type { RootState } from '../index'

export const fetchGitAuth = createAsyncThunk('git/fetchGitAuth', async () => api.githubAuth())
export const fetchGitRepos = createAsyncThunk('git/fetchGitRepos', async () => api.githubRepos())
export const connectGit = createAsyncThunk('git/connectGit', async (token: string) => api.connectGitHub(token))
export const pushWiki = createAsyncThunk('git/pushWiki', async (url: string) => api.gitSetup(url))
export const disconnectGit = createAsyncThunk('git/disconnectGit', async () => api.disconnectGitHub())

interface GitState {
    auth: GitHubIdentity | null
    repos: GitHubRepo[] | null
    loading: boolean
    connecting: boolean
    pushing: boolean
    // Server-rejected attempts carry a message worth showing (e.g. an
    // invalid token, a public repo); transport failures fall back to a
    // fixed one.
    error: string | null
}

const initialState: GitState = {
    auth: null,
    repos: null,
    loading: false,
    connecting: false,
    pushing: false,
    error: null
}

export const gitSlice = createSlice({
    name: 'git',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchGitAuth.pending, (s) => {
                s.loading = true
            })
            .addCase(fetchGitAuth.fulfilled, (s, a: PayloadAction<GitHubIdentity>) => {
                s.auth = a.payload
                s.loading = false
            })
            .addCase(fetchGitAuth.rejected, (s) => {
                // Not connected is a normal state, not an error.
                s.auth = null
                s.loading = false
                s.error = null
            })
            .addCase(fetchGitRepos.fulfilled, (s, a: PayloadAction<{ repos: GitHubRepo[] }>) => {
                s.repos = a.payload.repos
            })
            .addCase(connectGit.pending, (s) => {
                s.connecting = true
                s.error = null
            })
            .addCase(connectGit.fulfilled, (s, a: PayloadAction<GitHubIdentity>) => {
                s.auth = a.payload
                s.connecting = false
            })
            .addCase(connectGit.rejected, (s, a) => {
                s.connecting = false
                s.error = a.error.message ?? 'could not connect to GitHub'
            })
            .addCase(pushWiki.pending, (s) => {
                s.pushing = true
                s.error = null
            })
            .addCase(pushWiki.fulfilled, (s, a: PayloadAction<{ ok: boolean; error?: string }>) => {
                s.pushing = false
                // gitSetup reports business failures as ok:false — surface
                // the server message like a rejected connection.
                s.error = a.payload.ok ? null : (a.payload.error ?? 'could not push the wiki')
            })
            .addCase(pushWiki.rejected, (s) => {
                s.pushing = false
                s.error = 'could not reach the server'
            })
            .addCase(disconnectGit.fulfilled, (s) => {
                s.auth = null
                s.repos = null
                s.error = null
            })
            .addCase(disconnectGit.rejected, (s) => {
                s.error = 'could not disconnect from GitHub'
            })
    }
})

export const selectGitAuth = (s: RootState) => s.git.auth
export const selectGitRepos = (s: RootState) => s.git.repos
export const selectGitLoading = (s: RootState) => s.git.loading
export const selectGitConnecting = (s: RootState) => s.git.connecting
export const selectGitPushing = (s: RootState) => s.git.pushing
export const selectGitError = (s: RootState) => s.git.error
