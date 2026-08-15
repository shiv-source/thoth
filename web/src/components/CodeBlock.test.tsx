import { screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithStore } from '../test/renderWithStore'
import { CodeBlock } from './CodeBlock'

const mocks = vi.hoisted(() => ({ codeToHtml: vi.fn() }))
vi.mock('shiki', () => ({ codeToHtml: mocks.codeToHtml }))

describe('CodeBlock', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('shows the plain fallback until highlighting resolves', () => {
        mocks.codeToHtml.mockReturnValue(new Promise(() => {})) // never resolves
        renderWithStore(<CodeBlock code="const x = 1" lang="ts" />)
        expect(screen.getByText('const x = 1')).toBeInTheDocument()
    })

    it('replaces the fallback with highlighted HTML once ready', async () => {
        mocks.codeToHtml.mockResolvedValue('<pre class="shiki"><code><span>hl</span></code></pre>')
        renderWithStore(<CodeBlock code="const x = 1" lang="ts" />)
        expect(await screen.findByText('hl')).toBeInTheDocument()
        expect(mocks.codeToHtml).toHaveBeenCalledWith('const x = 1', expect.objectContaining({ lang: 'ts' }))
    })

    it('highlights identical code+language only once across mounts', async () => {
        mocks.codeToHtml.mockResolvedValue('<pre class="shiki"><code><span>hl</span></code></pre>')
        // Unique snippet: the module-level cache persists across tests in
        // this file, so a fresh key proves the within-test behavior.
        const first = renderWithStore(<CodeBlock code="const y = 2" lang="ts" />)
        await screen.findByText('hl')
        first.unmount()
        const second = renderWithStore(<CodeBlock code="const y = 2" lang="ts" />)
        expect(await screen.findByText('hl')).toBeInTheDocument()
        second.unmount()
        expect(mocks.codeToHtml).toHaveBeenCalledTimes(1)
    })
})
