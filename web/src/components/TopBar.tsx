export function TopBar({ title, onOpenSettings }: {
  title: string
  onOpenSettings: () => void
}) {
  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-line bg-surface px-4">
      <h1 className="truncate text-sm font-medium text-ink">{title}</h1>
      <div className="flex shrink-0 items-center gap-1">
        <button onClick={onOpenSettings} aria-label="Settings" title="Settings"
          className="rounded-lg p-2 text-subtle transition hover:bg-raised hover:text-ink">
          <GearIcon />
        </button>
      </div>
    </header>
  )
}

function GearIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"
      className="h-4 w-4">
      <circle cx="10" cy="10" r="3" />
      <path d="M19 10a9 9 0 0 0-.2-1.8l-2.1-.6a7 7 0 0 0-1.1-1.9l.6-2.1a9 9 0 0 0-3.2-1.8l-1.6 1.5a7 7 0 0 0-1.4-.2 7 7 0 0 0-1.4.2l-1.6-1.5A9 9 0 0 0 3.8 3.6l.6 2.1a7 7 0 0 0-1.1 1.9l-2.1.6A9 9 0 0 0 1 10c0 .6.1 1.2.2 1.8l2.1.6a7 7 0 0 0 1.1 1.9l-.6 2.1a9 9 0 0 0 3.2 1.8l1.6-1.5c.4.1 1 .2 1.4.2s1-.1 1.4-.2l1.6 1.5a9 9 0 0 0 3.2-1.8l-.6-2.1a7 7 0 0 0 1.1-1.9l2.1-.6A9 9 0 0 0 19 10Z" />
    </svg>
  )
}
