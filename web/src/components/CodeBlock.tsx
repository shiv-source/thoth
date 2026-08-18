import { useEffect, useState } from 'react'
import { codeToHtml } from 'shiki'
import { CopyButton } from './CopyButton'

// The cache is module-level so repeated code blocks across a conversation
// highlight once; capped so it cannot grow unbounded.
const cache = new Map<string, string>()
const CACHE_MAX = 200

function highlight(code: string, lang: string): Promise<string> {
    const key = `${lang}|${code}`
    const hit = cache.get(key)
    if (hit !== undefined) return Promise.resolve(hit)
    return codeToHtml(code, { lang: lang || 'text', theme: 'github-light' }).then((html) => {
        if (cache.size >= CACHE_MAX) cache.clear()
        cache.set(key, html)
        return html
    })
}

// CodeBlock renders one fenced code block through Shiki with a copy button.
// Until the async highlight resolves it shows the plain pre/code fallback;
// the highlighted HTML replaces it (Shiki output is escaped, safe to inject).
export function CodeBlock({ code, lang }: { code: string; lang: string }) {
    const [html, setHtml] = useState<string | null>(() => cache.get(`${lang}|${code}`) ?? null)

    useEffect(() => {
        if (html !== null) return
        let alive = true
        highlight(code, lang)
            .then((h) => {
                if (alive) setHtml(h)
            })
            .catch(() => {
                if (alive) setHtml('') // fall back to the plain pre forever
            })
        return () => {
            alive = false
        }
    }, [code, lang, html])

    return (
        <div className="group relative">
            <CopyButton
                text={code}
                label="Copy code"
                toast="Code copied to clipboard"
                className="absolute right-2 top-2 z-10 rounded-md border border-line bg-surface/90 p-1.5 opacity-0 transition group-hover:opacity-100 focus-visible:opacity-100"
            />
            {html === null || html === '' ? (
                <pre className="overflow-x-auto rounded-lg bg-[#f6f8fa] p-3 text-xs text-[#1f2328]">
                    <code>{code}</code>
                </pre>
            ) : (
                <div
                    className="overflow-x-auto rounded-lg text-xs [&_pre]:m-0 [&_pre]:p-3 [&_pre]:bg-[#f6f8fa]"
                    dangerouslySetInnerHTML={{ __html: html }}
                />
            )}
        </div>
    )
}
