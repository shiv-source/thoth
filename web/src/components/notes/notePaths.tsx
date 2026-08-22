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
    return `/api/notes?path=${encodeURIComponent(path)}`
}
