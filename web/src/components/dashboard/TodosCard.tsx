import { useState } from 'react'
import { Badge, Card, Checkbox } from 'antd'

// Todo is one row of the todos list — the text and whether it's done. Until
// the todos endpoint lands, toggling is local state only.
export interface Todo {
    text: string
    done: boolean
}

// TodosCard is the Overview "Todo list" widget: the open todos as a
// checkable checklist with a count badge. Checking a row strikes it through.
export function TodosCard({ todos }: { todos: Todo[] }) {
    const [items, setItems] = useState(todos)
    const open = items.filter((t) => !t.done).length

    const toggle = (index: number) =>
        setItems((prev) => prev.map((t, i) => (i === index ? { ...t, done: !t.done } : t)))

    return (
        <Card size="small" title="Todo list" extra={<Badge count={open} color="var(--ant-color-fill-secondary)" />}>
            {items.length === 0 ? (
                <p className="py-2 text-sm text-subtle">Nothing on the list — nice.</p>
            ) : (
                <ul className="flex flex-col gap-1">
                    {items.map((t, i) => (
                        <li key={t.text}>
                            <Checkbox checked={t.done} onChange={() => toggle(i)} className="w-full">
                                <span className={t.done ? 'text-faint line-through' : 'text-ink'}>{t.text}</span>
                            </Checkbox>
                        </li>
                    ))}
                </ul>
            )}
            <p className="mt-3 text-xs text-subtle">mock data — todos/TODO.md</p>
        </Card>
    )
}
