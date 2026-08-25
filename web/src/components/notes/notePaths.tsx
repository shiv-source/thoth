// notePaths — the previewability rules shared by the note viewer and its
// body: which wiki paths are markdown notes, which are images, and how to
// build the raw-bytes attachment URL.

// isNotePath reports whether a wiki path is a previewable Markdown note
// (.md or .markdown, case-insensitive — matching wiki.IsMarkdownPath). The
// tree only lists markdown, but attachments (images, scripts, …) are indexed
// by filename and reachable by search or direct URL; those render as an image
// preview or a download instead of raw bytes as Markdown.
export function isNotePath(path: string): boolean {
    return /\.(?:md|markdown)$/i.test(path)
}

// isImagePath reports whether a wiki path is a previewable image attachment
// (.png/.jpg/.jpeg/.gif/.svg/.webp, case-insensitive — matching
// wiki.IsImagePath). Images render inline; every other attachment gets a
// download action.
export function isImagePath(path: string): boolean {
    return /\.(?:png|jpe?g|gif|svg|webp)$/i.test(path)
}

// noteUrl is the raw-bytes URL for an attachment: the server wraps markdown
// in JSON but serves any other path as raw bytes (images inline, everything
// else as a download), so an <img> or download link can point straight at it.
export function noteUrl(path: string): string {
    return `/api/v1/notes?path=${encodeURIComponent(path)}`
}

// stripFrontmatter removes a leading YAML frontmatter block from a note's
// content, mirroring the wiki's splitFrontmatter: an optional UTF-8 BOM, an
// opening `---` fence, then the first later line that is exactly `---` or
// `...` closes it. Returns the body after the closing fence; content without
// a leading fence (or with an unclosed one) is returned unchanged.
export function stripFrontmatter(content: string): string {
    let text = content
    if (text.charCodeAt(0) === 0xfeff) text = text.slice(1)
    const firstLineEnd = text.indexOf('\n')
    const firstLine = firstLineEnd === -1 ? text : text.slice(0, firstLineEnd)
    if (!/^---[ \t]*$/.test(firstLine)) return content

    let rest = firstLineEnd === -1 ? '' : text.slice(firstLineEnd + 1)
    for (;;) {
        const lineEnd = rest.indexOf('\n')
        const line = lineEnd === -1 ? rest : rest.slice(0, lineEnd)
        if (/^(?:---|\.\.\.)[ \t]*$/.test(line)) {
            return lineEnd === -1 ? '' : rest.slice(lineEnd + 1)
        }
        if (lineEnd === -1) return content // unclosed frontmatter — leave as-is
        rest = rest.slice(lineEnd + 1)
    }
}
