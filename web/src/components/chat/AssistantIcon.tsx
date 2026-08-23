import { LogoMark } from '../../shared/LogoMark'

// AssistantIcon is the small brand tile shown to the left of every
// assistant message — the same owl mark as the app logo, so the assistant
// reads as part of the product rather than a stock robot.
export function AssistantIcon() {
    return <LogoMark size={28} />
}
