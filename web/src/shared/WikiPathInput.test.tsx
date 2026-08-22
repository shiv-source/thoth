import { Form } from 'antd'
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../test/mockAxios'
import { renderWithStore } from '../test/renderWithStore'
import { WikiPathInput } from './WikiPathInput'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

// renderField mounts the input inside an antd Form so the value/onChange
// contract matches the real Settings form.
function renderField(initial: string) {
    return renderWithStore(
        <Form initialValues={{ wiki_path: initial }}>
            <Form.Item name="wiki_path">
                <WikiPathInput />
            </Form.Item>
        </Form>
    )
}

describe('WikiPathInput', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('keeps the value hand-editable', async () => {
        renderField('~/.thoth/wiki')
        const input = await screen.findByDisplayValue('~/.thoth/wiki')
        await userEvent.clear(input)
        await userEvent.type(input, '/tmp/notes')
        expect(screen.getByDisplayValue('/tmp/notes')).toBeInTheDocument()
    })

    it('opens the directory picker from the folder icon and selects a directory', async () => {
        mocks.get.mockImplementation((url: string) => {
            const param = new URLSearchParams(String(url).split('?')[1] ?? '')
            const path = param.get('path')
            if (path === '~') return Promise.resolve({ data: { dirs: ['/home/u/wiki', '/home/u/notes'] } })
            if (path === '/home/u/wiki') return Promise.resolve({ data: { dirs: ['/home/u/wiki/inbox'] } })
            return Promise.reject(new Error('unhandled ' + url))
        })

        renderField('~')
        await userEvent.click(await screen.findByLabelText('Choose wiki directory'))

        const dialog = await screen.findByRole('dialog')
        // The browser starts at the current value (~) and lists its
        // subdirectories.
        await within(dialog).findByText('wiki')
        await userEvent.click(within(dialog).getByText('wiki'))
        // Entering the directory loads its children.
        await within(dialog).findByText('inbox')
        await userEvent.click(within(dialog).getByRole('button', { name: 'Select this directory' }))

        expect(await screen.findByDisplayValue('/home/u/wiki')).toBeInTheDocument()
    })

    it('shows an error when the directory cannot be read', async () => {
        mocks.get.mockRejectedValueOnce(new Error('boom'))
        renderField('~/.thoth/wiki')
        await userEvent.click(await screen.findByLabelText('Choose wiki directory'))
        expect(await screen.findByText(/Cannot read this directory/)).toBeInTheDocument()
    })
})
