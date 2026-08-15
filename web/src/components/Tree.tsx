import { useCallback, useRef, useState, type KeyboardEvent } from 'react'
import { ChevronRight } from 'lucide-react'

export interface TreeProps<T> {
  nodes: T[]
  /** Stable unique key per node. */
  getKey: (n: T) => string
  getLabel: (n: T) => string
  isDir: (n: T) => boolean
  /** Children of a dir (empty for files). */
  getChildren: (n: T) => T[]
  /** Icon slot (e.g. folder/file icons); expanded is only meaningful for dirs. */
  renderIcon: (n: T, expanded: boolean) => React.ReactNode
  /** Optional trailing slot rendered right-aligned on rows (e.g. count badges). */
  renderTrailing?: (n: T) => React.ReactNode
  onSelect: (n: T) => void
  /** Key of the selected node (e.g. the open note). */
  selectedKey: string | null
  /** Controlled expansion (optional): when provided, Tree defers all
   *  expand/collapse decisions to the caller. */
  expandedKeys?: Set<string>
  onExpandedChange?: (next: Set<string>) => void
}

// Tree is a reusable, accessible folder tree. Folders start collapsed (the
// enterprise convention) and expand on the chevron click or ArrowRight.
// Keyboard: ArrowDown/Up move focus, ArrowRight/Left expand/collapse, Enter
// selects.
export function Tree<T>({
  nodes, getKey, getLabel, isDir, getChildren, renderIcon, renderTrailing,
  onSelect, selectedKey, expandedKeys, onExpandedChange,
}: TreeProps<T>) {
  const controlled = expandedKeys !== undefined
  // Flattened, focusable rows in visual order — roving tabIndex.
  const rows = useRef<T[]>([])
  const rowRefs = useRef(new Map<string, HTMLElement>())
  const [focusedKey, setFocusedKey] = useState<string | null>(null)
  const [internalExpanded, setInternalExpanded] = useState<Set<string>>(() => new Set())

  const expanded = controlled ? expandedKeys : internalExpanded

  const toggle = (n: T) => {
    const next = new Set(expanded)
    const key = getKey(n)
    if (next.has(key)) {
      next.delete(key)
    } else {
      next.add(key)
    }
    if (controlled) {
      onExpandedChange?.(next)
    } else {
      setInternalExpanded(next)
    }
  }

  const moveFocus = (dir: 1 | -1) => {
    setFocusedKey((current) => {
      const idx = rows.current.findIndex((n) => getKey(n) === current)
      const next = rows.current[idx + dir]
      if (!next) return current
      const key = getKey(next)
      rowRefs.current.get(key)?.focus()
      return key
    })
  }

  const onKeyDown = useCallback((e: KeyboardEvent<HTMLElement>, n: T) => {
    const dir = isDir(n)
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        moveFocus(1)
        break
      case 'ArrowUp':
        e.preventDefault()
        moveFocus(-1)
        break
      case 'ArrowRight':
        e.preventDefault()
        if (dir && !expanded.has(getKey(n))) toggle(n)
        break
      case 'ArrowLeft':
        e.preventDefault()
        if (dir && expanded.has(getKey(n))) toggle(n)
        break
      case 'Enter':
        e.preventDefault()
        onSelect(n)
        break
    }
  }, [expanded, isDir, getKey, onSelect])

  // Rebuild the flattened row list (only expanded dirs contribute children).
  const visible: T[] = []
  const walk = (list: T[]) => {
    for (const n of list) {
      visible.push(n)
      if (isDir(n) && expanded.has(getKey(n))) walk(getChildren(n))
    }
  }
  walk(nodes)
  rows.current = visible

  const render = (list: T[], depth: number): React.ReactNode => (
    <ul role={depth === 0 ? 'tree' : 'group'} className={depth === 0 ? 'space-y-px text-sm' : 'ml-3 space-y-px border-l border-line pl-1'}>
      {list.map((n) => {
        const dir = isDir(n)
        const key = getKey(n)
        const open = dir && expanded.has(key)
        const selected = selectedKey === key
        return (
          <li key={key} role="treeitem" aria-expanded={dir ? open : undefined} aria-selected={selected}>
            <div
              ref={(el) => {
                if (el) {
                  rowRefs.current.set(key, el)
                } else {
                  rowRefs.current.delete(key)
                }
              }}
              tabIndex={focusedKey === key ? 0 : -1}
              onFocus={() => setFocusedKey(key)}
              onKeyDown={(e) => onKeyDown(e, n)}
              onClick={() => (dir ? toggle(n) : onSelect(n))}
              className={`flex w-full cursor-pointer items-center gap-1 rounded-md px-1.5 py-1 transition ${
                selected ? 'bg-accent-soft font-medium text-accent' : 'text-ink hover:bg-raised'
              }`}
            >
              {dir ? (
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); toggle(n) }}
                  aria-label={`${open ? 'Collapse' : 'Expand'} ${getLabel(n)}`}
                  className="flex h-5 w-5 shrink-0 items-center justify-center rounded text-subtle transition hover:text-ink"
                >
                  <ChevronRight className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-90' : ''}`} />
                </button>
              ) : (
                <span className="w-5 shrink-0" />
              )}
              <span className="w-5 shrink-0 text-subtle">{renderIcon(n, open)}</span>
              <span className="min-w-0 flex-1 truncate">{getLabel(n)}</span>
              {renderTrailing?.(n)}
            </div>
            {dir && open && render(getChildren(n), depth + 1)}
          </li>
        )
      })}
    </ul>
  )

  return <>{render(nodes, 0)}</>
}
