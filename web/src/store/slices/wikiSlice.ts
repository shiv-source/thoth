import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api, type TreeNode } from '../../api/client'
import type { RootState } from '../index'

export const fetchTree = createAsyncThunk('wiki/fetchTree', async () => api.tree())

interface WikiState {
    nodes: TreeNode[] | null
    loading: boolean
    error: string | null
}

const initialState: WikiState = { nodes: null, loading: true, error: null }

export const wikiSlice = createSlice({
    name: 'wiki',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchTree.pending, (s) => {
                s.loading = true
                s.error = null
            })
            .addCase(fetchTree.fulfilled, (s, a: PayloadAction<{ nodes: TreeNode[] }>) => {
                s.nodes = a.payload.nodes
                s.loading = false
            })
            .addCase(fetchTree.rejected, (s) => {
                s.loading = false
                s.error = 'could not load the wiki tree'
            })
    }
})

export const selectWikiNodes = (s: RootState) => s.wiki.nodes
export const selectWikiLoading = (s: RootState) => s.wiki.loading
export const selectWikiError = (s: RootState) => s.wiki.error

// collectTreeInfo walks the tree once: every dir key (for the
// expand/collapse-all toggle) plus the recursive file count per dir (for
// the hover tooltips). Shared by NotesPage and WikiTree.
export function collectTreeInfo(nodes: TreeNode[]): { allDirs: Set<string>; fileCounts: Map<string, number> } {
    const dirs = new Set<string>()
    const counts = new Map<string, number>()
    const walk = (list: TreeNode[]): number => {
        let files = 0
        for (const n of list) {
            if (n.is_dir) {
                dirs.add(n.path)
                const sub = walk(n.children ?? [])
                counts.set(n.path, sub)
                files += sub
            } else {
                files++
            }
        }
        return files
    }
    walk(nodes)
    return { allDirs: dirs, fileCounts: counts }
}
