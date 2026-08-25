import { ArrowDownOutlined, ArrowUpOutlined } from '@ant-design/icons'

// UsageTokens renders the input/output token counts with directional arrows —
// the shared "tokens in / tokens out" motif for per-message headers and the
// conversation total. The arrows carry the accessible name; the numbers keep
// tabular alignment.
export function UsageTokens({ input, output }: { input: number; output: number }) {
    return (
        <>
            <span className="inline-flex items-center gap-1">
                <ArrowDownOutlined aria-label="input tokens" className="text-[10px]" />
                <span className="tabular-nums">{input.toLocaleString()}</span>
            </span>
            <span aria-hidden="true">·</span>
            <span className="inline-flex items-center gap-1">
                <ArrowUpOutlined aria-label="output tokens" className="text-[10px]" />
                <span className="tabular-nums">{output.toLocaleString()}</span>
            </span>
        </>
    )
}
