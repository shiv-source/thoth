import { describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import { runDoctor, selectDoctorChecks, selectDoctorError, selectDoctorRunning } from '../index'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

describe('doctorSlice', () => {
    it('runs the checks and stores them', async () => {
        stubAPI(mocks, {
            '/api/v1/doctor': () => ({
                checks: [
                    { name: 'claude', ok: true, message: 'found' },
                    { name: 'wiki', ok: false, message: 'missing' }
                ]
            })
        })
        const store = makeStore()
        await store.dispatch(runDoctor())
        expect(selectDoctorRunning(store.getState())).toBe(false)
        expect(selectDoctorChecks(store.getState())).toEqual([
            { name: 'claude', ok: true, message: 'found' },
            { name: 'wiki', ok: false, message: 'missing' }
        ])
    })

    it('records an error when the check run fails', async () => {
        stubAPI(mocks, {})
        const store = makeStore()
        await store.dispatch(runDoctor()).catch(() => {})
        expect(selectDoctorError(store.getState())).toBe('could not run the doctor checks')
    })
})
