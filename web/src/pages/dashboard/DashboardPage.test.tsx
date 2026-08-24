import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../../test/mockAxios'
import { renderWithStore } from '../../test/renderWithStore'
import { DashboardPage } from './DashboardPage'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

// jsdom has no 2D canvas, so the charts can't run here. Mock at the seam
// the components actually use — the react-chartjs-2 wrappers render the
// canvas (with the aria-label they're given); chart.js only needs its
// registration exports to exist.
vi.mock('chart.js', () => {
    class ChartStub {
        static register() {}
        update() {}
        destroy() {}
    }
    return {
        ArcElement: vi.fn(),
        BarController: vi.fn(),
        BarElement: vi.fn(),
        CategoryScale: vi.fn(),
        DoughnutController: vi.fn(),
        Filler: vi.fn(),
        LinearScale: vi.fn(),
        LineController: vi.fn(),
        LineElement: vi.fn(),
        PointElement: vi.fn(),
        Tooltip: vi.fn(),
        register: () => {},
        Chart: ChartStub
    }
})

vi.mock('react-chartjs-2', () => {
    const Canvas = (props: { 'aria-label'?: string; role?: string }) => (
        <canvas role={props.role ?? 'img'} aria-label={props['aria-label']} />
    )
    return { Bar: Canvas, Line: Canvas, Doughnut: Canvas }
})

const conversations = {
    conversations: [
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1', title: 'Today chat', created_at: new Date().toISOString() },
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2', title: 'Old chat', created_at: new Date().toISOString() }
    ]
}

function renderDashboard() {
    return renderWithStore(<DashboardPage onOpenSettings={vi.fn()} />)
}

describe('DashboardPage', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        mocks.get.mockResolvedValue({ data: conversations })
        // setSystemTime alone (no fake timers) keeps Date deterministic
        // without breaking waitFor/findBy.
        vi.setSystemTime(new Date('2026-08-15T09:30:00'))
    })

    afterEach(() => {
        vi.useRealTimers()
        window.history.pushState(null, '', '/')
    })

    it('renders the time-based greeting and today date', async () => {
        renderDashboard()
        expect(screen.getByText('Good morning')).toBeInTheDocument()
        expect(screen.getByText(/Saturday, August 15/)).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('renders the mock tiles with their dummy data', async () => {
        renderDashboard()
        expect(screen.getByText('3 captures waiting')).toBeInTheDocument()
        // "Standup" appears in the resume strip and the Today timeline.
        expect(screen.getAllByText('Standup').length).toBeGreaterThanOrEqual(1)
        expect(screen.getByText(/Wire the todos tile/)).toBeInTheDocument()
        expect(screen.getByText('links/bookmarks.md')).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('renders the stat tiles from mock data', async () => {
        renderDashboard()
        expect(screen.getByText('128')).toBeInTheDocument()
        expect(screen.getByText('2h ago')).toBeInTheDocument()
        expect(screen.getByText('Last sync')).toBeInTheDocument()
        // The KPI tile label and the "Needs attention" row both carry it.
        expect(screen.getByText('Open todos')).toBeInTheDocument()
        expect(screen.getByText('2 open todos')).toBeInTheDocument()
        // Flush the conversations fetch resolution inside act, so its store
        // update can't land after the test body (act() warning).
        await waitFor(() => expect(mocks.get).toHaveBeenCalled())
    })

    it('renders the chart canvases with aria-labels', async () => {
        renderDashboard()
        // System time is Aug 15 2026, so the weekly series spans Aug 9–15.
        expect(screen.getByRole('img', { name: /Notes created per day.*Aug 9 to Aug 15/ })).toBeInTheDocument()
        expect(screen.getByRole('img', { name: /Chat messages per day for the last 14 days/ })).toBeInTheDocument()
        expect(
            screen.getByRole('img', { name: /Notes by kind: Meetings 24, Captures 18, Knowledge 63, Links 23/ })
        ).toBeInTheDocument()
        expect(
            screen.getByRole('img', { name: /Notes per wiki folder: knowledge 63, links 23, meetings 24, capture 18/ })
        ).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('renders the notes-by-kind legend with the series labels', async () => {
        renderDashboard()
        expect(screen.getAllByText('Meetings').length).toBeGreaterThanOrEqual(1)
        expect(screen.getAllByText('Knowledge').length).toBeGreaterThanOrEqual(1)
        expect(screen.getAllByText('Links').length).toBeGreaterThanOrEqual(1)
        // "Captures" appears twice: the KPI tile label and the legend item.
        expect(screen.getAllByText('Captures').length).toBeGreaterThanOrEqual(2)
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('routes a tag chip to the search view', async () => {
        renderDashboard()
        window.history.pushState(null, '', '/dashboard')
        fireEvent.click(screen.getByRole('button', { name: '#go' }))
        expect(window.location.pathname).toBe('/search')
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('shows real recent chats in the resume strip and navigates on click', async () => {
        renderDashboard()
        expect(await screen.findByText('Today chat')).toBeInTheDocument()

        window.history.pushState(null, '', '/dashboard')
        fireEvent.click(screen.getByText('Today chat'))
        expect(window.location.pathname).toBe('/chat/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
    })

    it('opens a recent note from the resume strip in the notes view', async () => {
        renderDashboard()
        window.history.pushState(null, '', '/dashboard')
        fireEvent.click(screen.getByText('Renovate GitHub action'))
        expect(window.location.pathname).toBe('/notes/knowledge/renovate-github-action.md')
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('captures quick-capture text into the inbox and toasts the path', async () => {
        renderDashboard()
        fireEvent.change(screen.getByRole('textbox', { name: 'Quick capture' }), {
            target: { value: 'Check the npm audit' }
        })
        fireEvent.click(screen.getByRole('button', { name: /Capture/ }))
        expect(await screen.findByText('Captured to inbox/check-the-npm-audit.md (mock)')).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('routes the quick actions to their views', async () => {
        renderDashboard()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled())

        fireEvent.click(screen.getByRole('button', { name: /Ask the wiki/ }))
        expect(window.location.pathname).toBe('/search')

        fireEvent.click(screen.getByRole('button', { name: /New note/ }))
        expect(window.location.pathname).toBe('/notes')

        fireEvent.click(screen.getByRole('button', { name: /New chat/ }))
        expect(window.location.pathname).toBe('/chat')
    })
})
