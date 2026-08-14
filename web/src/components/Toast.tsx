import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'

export type ToastKind = 'success' | 'error'

interface ToastItem {
  id: number
  message: string
  kind: ToastKind
}

interface ToastContextValue {
  toast: (message: string, kind?: ToastKind) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

// ToastProvider renders a fixed bottom-center stack (z-50). Each toast
// auto-dismisses after 3 s and closes on click.
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const nextId = useRef(0)
  const timers = useRef(new Map<number, ReturnType<typeof setTimeout>>())

  const dismiss = useCallback((id: number) => {
    const timer = timers.current.get(id)
    if (timer !== undefined) {
      clearTimeout(timer)
      timers.current.delete(id)
    }
    setToasts((cur) => cur.filter((t) => t.id !== id))
  }, [])

  const toast = useCallback((message: string, kind: ToastKind = 'success') => {
    const id = nextId.current++
    setToasts((cur) => [...cur, { id, message, kind }])
    timers.current.set(id, setTimeout(() => dismiss(id), 3000))
  }, [dismiss])

  // Every timer is cleaned up when the provider unmounts (tests unmount mid-toast).
  useEffect(() => () => {
    for (const timer of timers.current.values()) clearTimeout(timer)
    timers.current.clear()
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div aria-live="polite" className="pointer-events-none fixed inset-x-0 bottom-6 z-50 flex flex-col items-center gap-2 px-4">
        {toasts.map((t) => (
          <button key={t.id} type="button" onClick={() => dismiss(t.id)}
            className="pointer-events-auto flex w-full max-w-sm items-center gap-2.5 rounded-xl border border-line bg-surface px-4 py-2.5 text-left text-sm text-ink shadow-md transition hover:bg-raised">
            <span aria-hidden="true" className={`h-2 w-2 shrink-0 rounded-full ${t.kind === 'success' ? 'bg-emerald-500' : 'bg-red-500'}`} />
            {t.message}
          </button>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within a ToastProvider')
  return ctx
}
