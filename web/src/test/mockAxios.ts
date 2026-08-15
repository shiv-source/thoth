import { vi } from 'vitest'

export interface AxiosMethodMocks {
  get: ReturnType<typeof vi.fn>
  post: ReturnType<typeof vi.fn>
  put: ReturnType<typeof vi.fn>
  delete: ReturnType<typeof vi.fn>
}

// axiosModuleMock is the factory for vi.mock('axios'). vi.mock/vi.hoisted
// cannot live inside an imported helper (hoisting is per test file), so each
// test file declares this AT THE TOP LEVEL:
//
//   const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
//   vi.mock('axios', () => axiosModuleMock(mocks))
//
export function axiosModuleMock(mocks: AxiosMethodMocks) {
  return {
    default: {
      create: () => ({
        get: mocks.get,
        post: mocks.post,
        put: mocks.put,
        delete: mocks.delete,
      }),
      isAxiosError: (e: unknown) => !!(e && typeof e === 'object' && (e as { isAxiosError?: boolean }).isAxiosError === true),
    },
  }
}

// stubAPI wires the mocks to the handlers, keyed by "METHOD /path" (with a
// plain path fallback). Handlers return the response BODY (axios wraps it as
// `{ data }`), so a handler that used to return a Response now returns the
// parsed object directly.
export function stubAPI(mocks: AxiosMethodMocks, handlers: Record<string, () => unknown>) {
  const respond = (method: string, url: string) => {
    const make = handlers[`${method} ${url}`] ?? handlers[url]
    if (!make) {
      return Promise.reject(Object.assign(new Error(`unhandled ${method} ${url}`), {
        isAxiosError: true,
        response: { status: 500, statusText: 'Internal Server Error' },
      }))
    }
    return Promise.resolve({ data: make() })
  }
  const typed = mocks as unknown as {
    get: ReturnType<typeof vi.fn<(url: string) => Promise<unknown>>>
    post: ReturnType<typeof vi.fn<(url: string, body?: unknown) => Promise<unknown>>>
    put: ReturnType<typeof vi.fn<(url: string, body?: unknown) => Promise<unknown>>>
    delete: ReturnType<typeof vi.fn<(url: string) => Promise<unknown>>>
  }
  typed.get.mockImplementation((url) => respond('GET', url))
  typed.post.mockImplementation((url) => respond('POST', url))
  typed.put.mockImplementation((url) => respond('PUT', url))
  typed.delete.mockImplementation((url) => respond('DELETE', url))
  return mocks
}

// axiosError builds a rejection value shaped like an axios error response —
// for tests that exercise error paths (e.g. connectGitHub 400 handling).
export function axiosError(status: number, body: unknown) {
  return Object.assign(new Error(`${status}`), {
    isAxiosError: true,
    response: { status, statusText: String(status), data: body },
  })
}
