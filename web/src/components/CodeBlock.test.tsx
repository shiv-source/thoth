import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithStore } from '../test/renderWithStore'
import { ToastProvider } from './Toast'
import { CodeBlock } from './CodeBlock'

const mocks = vi.hoisted(() => ({ codeToHtml: vi.fn() }))
vi.mock('shiki', () => ({ codeToHtml: mocks.codeToHtml }))

function renderBlock(code = 'const x = 1', lang = 'ts') {
    return renderWithStore(
        <ToastProvider>
            <CodeBlock code={code} lang={lang} />
        </ToastProvider>
    )
}

describe('CodeBlock', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('shows the plain fallback until highlighting resolves', () => {
        mocks.codeToHtml.mockReturnValue(new Promise(() => {})) // never resolves
        renderBlock()
        expect(screen.getByText('const x = 1')).toBeInTheDocument()
    })

    it('replaces the fallback with highlighted HTML once ready', async () => {
        mocks.codeToHtml.mockResolvedValue('<pre class="shiki"><code><span>hl</span></code></pre>')
        renderBlock()
        expect(await screen.findByText('hl')).toBeInTheDocument()
        expect(mocks.codeToHtml).toHaveBeenCalledWith('const x = 1', expect.objectContaining({ lang: 'ts' }))
    })

    it('highlights identical code+language only once across mounts', async () => {
        mocks.codeToHtml.mockResolvedValue('<pre class="shiki"><code><span>hl</span></code></pre>')
        // Unique snippet: the module-level cache persists across tests in
        // this file, so a fresh key proves the within-test behavior.
        const first = renderBlock('const y = 2')
        await screen.findByText('hl')
        first.unmount()
        const second = renderBlock('const y = 2')
        expect(await screen.findByText('hl')).toBeInTheDocument()
        second.unmount()
        expect(mocks.codeToHtml).toHaveBeenCalledTimes(1)
    })

    it('copies the raw code via the copy button', async () => {
        mocks.codeToHtml.mockResolvedValue('<pre class="shiki"><code><span>hl</span></code></pre>')
        const writeText = vi.fn().mockResolvedValue(undefined)
        Object.assign(navigator, { clipboard: { writeText } })
        renderBlock('const x = 1')
        await screen.findByText('hl')

        await userEvent.click(screen.getByRole('button', { name: 'Copy code' }))
        expect(writeText).toHaveBeenCalledWith('const x = 1')
        expect(await screen.findByText('Code copied to clipboard')).toBeInTheDocument()
    })
})
