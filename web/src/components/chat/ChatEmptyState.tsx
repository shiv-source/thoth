import { Button } from 'antd'
import { Logo } from '../../shared/Logo'

// Suggested prompts for the empty chat — clicking one starts a conversation
// with that question (the empty state only shows with no messages, so a send
// is the natural continuation).
const PROMPTS = [
    "What did we decide in Tuesday's standup?",
    'Summarize my open todos',
    'Save this: the client approved the new roadmap'
]

// ChatEmptyState is the brand hero shown when a conversation has no messages
// yet: the logo lockup, a headline, and suggested prompt chips that send
// directly.
export function ChatEmptyState({ onSend }: { onSend: (text: string) => void }) {
    return (
        <div className="flex h-full flex-col items-center justify-center gap-6 text-center">
            <Logo size={52} />
            <div className="space-y-2">
                <h2 className="font-display text-2xl font-semibold tracking-tight text-heading">Ask anything</h2>
                <p className="mx-auto max-w-md text-sm leading-relaxed text-subtle">
                    Thoth reads your wiki and answers from your notes — no need to switch tools.
                </p>
            </div>
            <div className="flex max-w-xl flex-wrap justify-center gap-2">
                {PROMPTS.map((p) => (
                    <Button key={p} shape="round" onClick={() => onSend(p)}>
                        {p}
                    </Button>
                ))}
            </div>
        </div>
    )
}
