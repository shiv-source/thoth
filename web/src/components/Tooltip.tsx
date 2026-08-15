// Tooltip shows a styled bubble next to its children on hover/focus. Pure
// CSS (group-hover + group-focus-within) — no state, no dependencies; hidden
// via opacity so the text stays in the DOM for accessibility and tests.
// side="bottom" is for controls inside scroll containers: a top bubble would
// be clipped by the container's overflow.
export function Tooltip({ label, children, side = 'top', align = 'center' }: {
  label: string
  children: React.ReactNode
  side?: 'top' | 'bottom'
  /** "end" right-aligns the bubble to the control — for controls at a
   *  panel's right edge, where a centered bubble would spill outside. */
  align?: 'center' | 'end'
}) {
  const placement = side === 'top'
    ? 'bottom-full mb-1.5'
    : 'top-full mt-1.5'
  const alignment = align === 'end'
    ? 'left-auto right-0 translate-x-0'
    : 'left-1/2 -translate-x-1/2'
  return (
    <span className="group relative inline-flex">
      {children}
      <span role="tooltip"
        className={`pointer-events-none absolute z-50 whitespace-nowrap rounded-md bg-ink px-2 py-1 text-[11px] font-medium text-app opacity-0 shadow-md transition-opacity duration-100 group-hover:opacity-100 group-focus-within:opacity-100 ${placement} ${alignment}`}>
        {label}
      </span>
    </span>
  )
}
