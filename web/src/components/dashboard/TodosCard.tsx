import { Card, Listy, Progress } from 'antd'

// Todo is one row of the open-todos widget.
export interface Todo {
    text: string
    done: boolean
}

// TodosCard is the Overview "Open todos" widget: the checklist with a
// done-progress bar underneath.
export function TodosCard({ todos }: { todos: Todo[] }) {
    const done = todos.filter((t) => t.done).length

    return (
        <Card size="small" title="Open todos">
            <Listy
                items={todos}
                rowKey={(t) => t.text}
                className="divide-y divide-line"
                classNames={{ item: 'p-0!' }}
                itemRender={(t) => (
                    <div
                        className={`flex items-center py-1 text-sm ${t.done ? 'text-subtle line-through' : 'text-ink'}`}
                    >
                        <span
                            aria-hidden="true"
                            className={`mr-2.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px] ${
                                t.done ? 'border-accent bg-accent text-accent-ink' : 'border-line bg-app'
                            }`}
                        >
                            {t.done ? '✓' : ''}
                        </span>
                        {t.text}
                    </div>
                )}
            />
            <div className="mt-3 flex items-center gap-2">
                <Progress
                    percent={todos.length === 0 ? 0 : Math.round((done / todos.length) * 100)}
                    size="small"
                    className="flex-1"
                />
                <span className="shrink-0 text-xs text-subtle">
                    {done} of {todos.length} done
                </span>
            </div>
            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
        </Card>
    )
}
