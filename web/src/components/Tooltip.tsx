import * as RTooltip from '@radix-ui/react-tooltip'

// Tooltip is a thin wrapper over Radix UI's tooltip: accessible out of the
// box, and its collision handling flips/shifts the bubble so it never spills
// outside the viewport or gets clipped by scroll containers. The app's dark
// bubble design lives here; every hover hint in the app uses this component.
export function Tooltip({ label, children, side = 'top', align = 'center' }: {
  label: string
  children: React.ReactNode
  side?: 'top' | 'bottom' | 'left' | 'right'
  align?: 'center' | 'start' | 'end'
}) {
  return (
    <RTooltip.Provider delayDuration={150} skipDelayDuration={100}>
      <RTooltip.Root>
        <RTooltip.Trigger asChild>{children}</RTooltip.Trigger>
        <RTooltip.Portal>
          <RTooltip.Content
            side={side}
            align={align}
            sideOffset={6}
            className="z-50 animate-[tooltip-fade_100ms_ease-out] rounded-md bg-ink px-2 py-1 text-[11px] font-medium text-app shadow-md"
          >
            {label}
            <RTooltip.Arrow className="fill-ink" width={6} height={3} />
          </RTooltip.Content>
        </RTooltip.Portal>
      </RTooltip.Root>
    </RTooltip.Provider>
  )
}
