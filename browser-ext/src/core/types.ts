// The capture kinds the server accepts. `selection` is folded into `note` by
// draftToCapture — a selection is a note whose body carries the blockquote.
export type CaptureKind = 'bookmark' | 'note' | 'selection' | 'readlater'

// DraftKind adds the two popup-only kinds: a text selection (rendered as a
// quoted note) and the assistant summarize flow, which rides its own endpoint.
export type DraftKind = 'bookmark' | 'note' | 'selection' | 'readlater' | 'summarize'

export interface CaptureInput {
    kind: CaptureKind
    url?: string
    title?: string
    text?: string
    reason?: string
    tags?: string[]
    folder?: string
    category?: string
}

export interface CaptureResponse {
    path: string
    title: string
    type: string
}

export interface DuplicateCheck {
    exists: boolean
    path?: string
}

// Draft is the pre-capture state the popup edits before it is sent to the
// server. Context-menu clicks and the capture command write a draft; the popup
// reads it on open, lets the user confirm the fields, then posts it.
export interface Draft {
    kind: DraftKind
    url: string
    title: string
    text?: string
    reason?: string
    category?: string
    folder?: string
    tags?: string[]
    includePageText?: boolean
}
