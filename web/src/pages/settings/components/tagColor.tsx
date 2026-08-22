// tagColor maps the seeded tags to stable antd preset colors so the same
// tag reads the same everywhere in the table; unknown tags fall back to the
// neutral default.
export function tagColor(tag: string): string {
    const colors: Record<string, string> = {
        strongest: 'gold',
        flagship: 'blue',
        balanced: 'green',
        fastest: 'cyan',
        fast: 'cyan',
        advanced: 'purple',
        efficient: 'lime',
        reasoning: 'magenta',
        coding: 'geekblue',
        powerful: 'volcano',
        open: 'default'
    }
    return colors[tag] ?? 'default'
}
