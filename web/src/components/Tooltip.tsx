// Tooltip shows a styled bubble above its children on hover/focus. Pure CSS
// (group-hover + group-focus-within) — no state, no dependencies; hidden via
// opacity so the text stays in the DOM for accessibility and tests.
export function Tooltip({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <span className="group relative inline-flex">
      {children}
      <span role="tooltip"
        className="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 -translate-x-1/2 whitespace-nowrap rounded-md bg-ink px-2 py-1 text-[11px] font-medium text-app opacity-0 shadow-md transition-opacity duration-100 group-hover:opacity-100 group-focus-within:opacity-100">
        {label}
      </span>
    </span>
  )
}
