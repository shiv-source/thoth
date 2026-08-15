import { memo, useCallback, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { ChevronRight } from 'lucide-react'
import { Tooltip } from './Tooltip'

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
    /** Optional hover tooltip per node; return undefined for no tooltip. */
    renderTooltip?: (n: T) => string | undefined
    onSelect: (n: T) => void
    /** Key of the selected node (e.g. the open note). */
    selectedKey: string | null
    /** Controlled expansion (optional): when provided, Tree defers all
     *  expand/collapse decisions to the caller. */
    expandedKeys?: Set<string>
    onExpandedChange?: (next: Set<string>) => void
}

interface TreeRowProps<T> {
    node: T
    selected: boolean
    open: boolean
    focused: boolean
    tip: string | undefined
    getKey: (n: T) => string
    getLabel: (n: T) => string
    isDir: (n: T) => boolean
    renderIcon: (n: T, expanded: boolean) => React.ReactNode
    renderTrailing?: (n: T) => React.ReactNode
    onSelect: (n: T) => void
    toggle: (n: T) => void
    onKeyDown: (e: KeyboardEvent<HTMLElement>, n: T) => void
    registerRow: (key: string, el: HTMLElement | null) => void
    onFocusRow: (key: string) => void
}

// TreeRow receives only stable or primitive props (node identity, booleans,
// memo-stable accessors/callbacks), so parent re-renders that leave a row's
// inputs untouched — e.g. a selectedKey move — skip it entirely.
function TreeRowInner<T>({
    node,
    selected,
    open,
    focused,
    tip,
    getKey,
    getLabel,
    isDir,
    renderIcon,
    renderTrailing,
    onSelect,
    toggle,
    onKeyDown,
    registerRow,
    onFocusRow
}: TreeRowProps<T>) {
    const dir = isDir(node)
    const key = getKey(node)
    const row = (
        <div
            ref={(el) => registerRow(key, el)}
            tabIndex={focused ? 0 : -1}
            onFocus={() => onFocusRow(key)}
            onKeyDown={(e) => onKeyDown(e, node)}
            onClick={() => (dir ? toggle(node) : onSelect(node))}
            className={`flex w-full cursor-pointer items-center gap-1 rounded-md px-1.5 py-1 transition ${
                selected ? 'bg-accent-soft font-medium text-accent' : 'text-ink hover:bg-raised'
            }`}
        >
            {dir ? (
                <button
                    type="button"
                    onClick={(e) => {
                        e.stopPropagation()
                        toggle(node)
                    }}
                    aria-label={`${open ? 'Collapse' : 'Expand'} ${getLabel(node)}`}
                    className="flex h-5 w-5 shrink-0 items-center justify-center rounded text-subtle transition hover:text-ink"
                >
                    <ChevronRight className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-90' : ''}`} />
                </button>
            ) : (
                <span className="w-5 shrink-0" />
            )}
            <span className="w-5 shrink-0 text-subtle">{renderIcon(node, open)}</span>
            <span className="min-w-0 flex-1 truncate">{getLabel(node)}</span>
            {renderTrailing?.(node)}
        </div>
    )
    return tip ? <Tooltip label={tip}>{row}</Tooltip> : row
}

// The cast preserves TreeRow's generic signature through React.memo.
const TreeRow = memo(TreeRowInner) as typeof TreeRowInner

// Tree is a reusable, accessible folder tree. Folders start collapsed (the
// enterprise convention) and expand on the chevron click or ArrowRight.
// Keyboard: ArrowDown/Up move focus, ArrowRight/Left expand/collapse, Enter
// selects.
export function Tree<T>({
    nodes,
    getKey,
    getLabel,
    isDir,
    getChildren,
    renderIcon,
    renderTrailing,
    renderTooltip,
    onSelect,
    selectedKey,
    expandedKeys,
    onExpandedChange
}: TreeProps<T>) {
    const controlled = expandedKeys !== undefined
    // Flattened, focusable rows in visual order — roving tabIndex.
    const rows = useRef<T[]>([])
    const rowRefs = useRef(new Map<string, HTMLElement>())
    const [focusedKey, setFocusedKey] = useState<string | null>(null)
    const [internalExpanded, setInternalExpanded] = useState<Set<string>>(() => new Set())

    const expanded = controlled ? expandedKeys : internalExpanded
    // Latest-value ref so toggle/onKeyDown stay memo-stable across expansion
    // changes (the same pattern as rows.current below).
    const expandedRef = useRef(expanded)
    expandedRef.current = expanded

    const toggle = useCallback(
        (n: T) => {
            const cur = expandedRef.current
            const next = new Set(cur)
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
        },
        [controlled, onExpandedChange, getKey]
    )

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

    const onKeyDown = useCallback(
        (e: KeyboardEvent<HTMLElement>, n: T) => {
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
                    if (dir && !expandedRef.current.has(getKey(n))) toggle(n)
                    break
                case 'ArrowLeft':
                    e.preventDefault()
                    if (dir && expandedRef.current.has(getKey(n))) toggle(n)
                    break
                case 'Enter':
                    e.preventDefault()
                    onSelect(n)
                    break
            }
        },
        [isDir, getKey, onSelect, toggle]
    )

    const registerRow = useCallback((key: string, el: HTMLElement | null) => {
        if (el) {
            rowRefs.current.set(key, el)
        } else {
            rowRefs.current.delete(key)
        }
    }, [])

    const onFocusRow = useCallback((key: string) => setFocusedKey(key), [])

    // Rebuild the flattened row list (only expanded dirs contribute children).
    const visible = useMemo(() => {
        const out: T[] = []
        const walk = (list: T[]) => {
            for (const n of list) {
                out.push(n)
                if (isDir(n) && expanded.has(getKey(n))) walk(getChildren(n))
            }
        }
        walk(nodes)
        return out
    }, [nodes, expanded, isDir, getKey, getChildren])
    rows.current = visible

    const render = (list: T[], depth: number): React.ReactNode => (
        <ul
            role={depth === 0 ? 'tree' : 'group'}
            className={depth === 0 ? 'space-y-px text-sm' : 'ml-3 space-y-px border-l border-line pl-1'}
        >
            {list.map((n) => {
                const dir = isDir(n)
                const key = getKey(n)
                const open = dir && expanded.has(key)
                return (
                    <li
                        key={key}
                        role="treeitem"
                        aria-expanded={dir ? open : undefined}
                        aria-selected={selectedKey === key}
                    >
                        <TreeRow
                            node={n}
                            selected={selectedKey === key}
                            open={open}
                            focused={focusedKey === key}
                            tip={renderTooltip?.(n)}
                            getKey={getKey}
                            getLabel={getLabel}
                            isDir={isDir}
                            renderIcon={renderIcon}
                            renderTrailing={renderTrailing}
                            onSelect={onSelect}
                            toggle={toggle}
                            onKeyDown={onKeyDown}
                            registerRow={registerRow}
                            onFocusRow={onFocusRow}
                        />
                        {dir && open && render(getChildren(n), depth + 1)}
                    </li>
                )
            })}
        </ul>
    )

    return <>{render(nodes, 0)}</>
}
