import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../test/mockAxios'
import { api } from './client'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

describe('api.search', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('parses results through zod', async () => {
        mocks.get.mockResolvedValue({
            data: { results: [{ path: 'meetings/a.md', title: 'A', kind: 'meeting', snippet: '…<mark>x</mark>…' }] }
        })
        const { results } = await api.search('x')
        expect(results[0]?.path).toBe('meetings/a.md')
    })

    it('rejects malformed payloads', async () => {
        mocks.get.mockResolvedValue({ data: { results: [{ path: 42 }] } })
        await expect(api.search('x')).rejects.toThrow()
    })
})

describe('api.listDirs', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('lists subdirectories for the wiki path picker', async () => {
        mocks.get.mockResolvedValue({ data: { dirs: ['/a/b', '/a/c'] } })
        const { dirs } = await api.listDirs('/a')
        expect(dirs).toEqual(['/a/b', '/a/c'])
        expect(mocks.get).toHaveBeenCalledWith('/api/fs/dirs?path=%2Fa')
    })
})

describe('api.models CRUD', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    const model = { id: 3, value: 'my-model', name: 'My Model', tag: 'test', provider: 'Vendor' }

    it('parses models grouped by provider through zod', async () => {
        mocks.get.mockResolvedValue({ data: { groups: [{ provider: 'Vendor', models: [model] }] } })
        const { groups } = await api.models()
        expect(groups[0]).toEqual({ provider: 'Vendor', models: [model] })
        expect(mocks.get).toHaveBeenCalledWith('/api/models')
    })

    it('rejects models missing the new fields', async () => {
        mocks.get.mockResolvedValue({
            data: { groups: [{ provider: 'Vendor', models: [{ value: 'x', label: 'X' }] }] }
        })
        await expect(api.models()).rejects.toThrow()
    })

    it('creates a model', async () => {
        mocks.post.mockResolvedValue({ data: model })
        const created = await api.createModel({ value: 'my-model', name: 'My Model' })
        expect(created).toEqual(model)
        expect(mocks.post).toHaveBeenCalledWith('/api/models', { value: 'my-model', name: 'My Model' })
    })

    it('updates a model', async () => {
        mocks.put.mockResolvedValue({ data: { ...model, name: 'Renamed' } })
        const updated = await api.updateModel(3, { value: 'my-model', name: 'Renamed' })
        expect(updated.name).toBe('Renamed')
        expect(mocks.put).toHaveBeenCalledWith('/api/models/3', { value: 'my-model', name: 'Renamed' })
    })

    it('deletes a model', async () => {
        mocks.delete.mockResolvedValue({ data: { ok: true } })
        await api.deleteModel(3)
        expect(mocks.delete).toHaveBeenCalledWith('/api/models/3')
    })
})

describe('api.getConversation and health', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('parses conversation history through zod', async () => {
        mocks.get.mockResolvedValue({
            data: {
                conversation: { id: 'c1', title: 'T', created_at: '2026-08-13T09:00:00Z' },
                messages: [
                    { id: 1, conversation_id: 'c1', role: 'user', content: 'hi', created_at: '2026-08-13T09:00:00Z' }
                ]
            }
        })
        const { messages, conversation } = await api.getConversation('c1')
        expect(conversation.id).toBe('c1')
        expect(messages[0]?.content).toBe('hi')
        expect(mocks.get).toHaveBeenCalledWith('/api/conversations/c1')
    })

    it('rejects unknown message roles', async () => {
        mocks.get.mockResolvedValue({
            data: {
                conversation: { id: 'c1', title: 'T', created_at: '2026-08-13T09:00:00Z' },
                messages: [
                    { id: 1, conversation_id: 'c1', role: 'system', content: 'hi', created_at: '2026-08-13T09:00:00Z' }
                ]
            }
        })
        await expect(api.getConversation('c1')).rejects.toThrow()
    })

    it('parses health and doctor responses', async () => {
        mocks.get.mockResolvedValueOnce({
            data: {
                status: 'ok',
                claude: { found: true, path: '/usr/local/bin/claude' },
                wiki: { path: '/tmp/wiki', exists: true },
                version: '1.2.3'
            }
        })
        mocks.get.mockResolvedValueOnce({
            data: { checks: [{ name: 'config', ok: true, message: 'parses' }] }
        })
        const health = await api.health()
        expect(health.claude.found).toBe(true)
        const { checks } = await api.doctor()
        expect(checks[0]?.name).toBe('config')
    })
})
