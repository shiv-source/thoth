// Tooltip shows a styled bubble next to its children on hover/focus. Pure
// CSS (group-hover + group-focus-within) — no state, no dependencies; hidden
// via opacity so the text stays in the DOM for accessibility and tests.
// side="bottom" is for controls inside scroll containers: a top bubble would
// be clipped by the container's overflow.
export function Tooltip({ label, children, side = 'top' }: {
  label: string
  children: React.ReactNode
  side?: 'top' | 'bottom'
}) {
  const placement = side === 'top'
    ? 'bottom-full mb-1.5'
    : 'top-full mt-1.5'
  return (
    <span className="group relative inline-flex">
      {children}
      <span role="tooltip"
        className={`pointer-events-none absolute left-1/2 z-50 -translate-x-1/2 whitespace-nowrap rounded-md bg-ink px-2 py-1 text-[11px] font-medium text-app opacity-0 shadow-md transition-opacity duration-100 group-hover:opacity-100 group-focus-within:opacity-100 ${placement}`}>
        {label}
      </span>
    </span>
  )
}
