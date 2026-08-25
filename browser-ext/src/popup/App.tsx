import { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Checkbox, Input, Segmented, Select, Space, Typography } from 'antd'
import { LinkOutlined } from '@ant-design/icons'
import { ThothClient, ThothError } from '../core/api'
import { refreshBadge } from '../core/badge'
import { BASE_URL_KEY, connectBaseUrl, ensureHostPermission, resolveBaseUrl, saveBaseUrl } from '../core/config'
import type { StorageLike } from '../core/config'
import {
    BOOKMARK_CATEGORIES,
    DEFAULT_CATEGORY,
    DEFAULT_NOTE_FOLDERS,
    draftToCapture,
    isHttpUrl,
    openNoteUrl,
    parseTags,
    sanitizeSingleLine,
    selectionToBody,
} from '../core/format'
import { capturePageText } from '../core/page'
import { clearDraft, loadDraft } from '../core/storage'
import type { CaptureResponse, DraftKind } from '../core/types'
import type { BrowserAPI } from '../core/webext'

type FieldName = 'title' | 'url' | 'text' | 'reason' | 'category' | 'folder' | 'tags' | 'fullpage'

// Which fields each capture kind shows. Selection maps to the Note tab but
// keeps its (read-only) source URL.
const KIND_FIELDS: Record<DraftKind, FieldName[]> = {
    note: ['title', 'text', 'folder', 'tags'],
    selection: ['title', 'url', 'text', 'folder', 'tags'],
    bookmark: ['title', 'url', 'category', 'reason', 'fullpage'],
    readlater: ['title', 'url', 'reason'],
    summarize: ['title', 'url', 'text', 'tags'],
}

const KIND_OPTIONS = [
    { label: 'Note', value: 'note' },
    { label: 'Bookmark', value: 'bookmark' },
    { label: 'Read later', value: 'readlater' },
    { label: 'Summarize', value: 'summarize' },
]

type Status = 'connecting' | 'ok' | 'down'

interface ResultState {
    tone: 'ok' | 'error'
    message: string
    path?: string
}

export interface PopupAppProps {
    storage: StorageLike
    ext: BrowserAPI
}

// getActivePageText reads the current tab's visible text via the scripting
// API. It works because opening the popup granted activeTab access.
async function getActivePageText(api: BrowserAPI): Promise<string> {
    const [tab] = await api.tabs.query({ active: true, currentWindow: true })
    if (!tab?.id) throw new ThothError('no active tab to read')
    const text = await capturePageText(api, tab.id)
    if (!text.trim()) throw new ThothError('could not read the page text')
    return text
}

export function PopupApp({ storage, ext }: PopupAppProps) {
    const [status, setStatus] = useState<Status>('connecting')
    const [statusText, setStatusText] = useState('Connecting…')
    const [baseUrlInput, setBaseUrlInput] = useState('')
    const [baseUrl, setBaseUrl] = useState('')
    const [client, setClient] = useState<ThothClient | null>(null)
    const [folders, setFolders] = useState<string[]>([])
    const [kind, setKind] = useState<DraftKind>('note')
    const [title, setTitle] = useState('')
    const [url, setUrl] = useState('')
    const [text, setText] = useState('')
    const [reason, setReason] = useState('')
    const [category, setCategory] = useState(DEFAULT_CATEGORY)
    const [folder, setFolder] = useState('inbox')
    const [tags, setTags] = useState('')
    const [includeFullPage, setIncludeFullPage] = useState(false)
    const [saving, setSaving] = useState(false)
    const [result, setResult] = useState<ResultState | null>(null)

    // Connect on mount: resolve the server (stored custom URL, else the
    // default ports), then load the pending capture draft and the folders.
    useEffect(() => {
        let cancelled = false
        void (async () => {
            setBaseUrlInput((await storage.get(BASE_URL_KEY)) ?? '')
            const found = await resolveBaseUrl(storage)
            if (cancelled) return
            if (found) {
                setBaseUrl(found)
                setClient(new ThothClient(found))
                setStatus('ok')
                setStatusText(`Connected to ${found}`)
            } else {
                setStatus('down')
                setStatusText('Thoth is not running — start it with `thoth serve`')
            }
            const draft = await loadDraft(storage)
            if (cancelled || !draft) return
            setKind(draft.kind)
            setTitle(draft.title ?? '')
            setUrl(draft.url ?? '')
            setReason(draft.reason ?? '')
            setTags(draft.tags?.join(', ') ?? '')
            setIncludeFullPage(draft.includePageText ?? false)
            setCategory(draft.category ?? DEFAULT_CATEGORY)
            if (draft.folder) setFolder(draft.folder)
            setText(
                draft.kind === 'selection'
                    ? selectionToBody(draft.text ?? '', draft.url ?? '', draft.title ?? '')
                    : (draft.text ?? '')
            )
        })()
        return () => {
            cancelled = true
        }
    }, [storage, ext])

    useEffect(() => {
        if (!client) return
        let cancelled = false
        void (async () => {
            let list: string[]
            try {
                list = await client.folders()
            } catch {
                list = []
            }
            if (!cancelled) setFolders(list.length ? list : [...DEFAULT_NOTE_FOLDERS])
        })()
        return () => {
            cancelled = true
        }
    }, [client])

    const connect = useCallback(async () => {
        await saveBaseUrl(storage, baseUrlInput)
        const granted = await ensureHostPermission(ext, baseUrlInput)
        if (!granted) {
            // Don't keep a stored URL the extension cannot reach — otherwise
            // the next popup open would silently fall back to discovery.
            await saveBaseUrl(storage, '')
            setBaseUrl('')
            setClient(null)
            setStatus('down')
            setStatusText('Permission to reach that server host was not granted')
            return
        }
        // A custom URL that is unreachable stays unreachable — never silently
        // fall back to the default port, or the status would lie about where
        // captures land.
        const found = await connectBaseUrl(baseUrlInput)
        if (found) {
            setBaseUrl(found)
            setClient(new ThothClient(found))
            setStatus('ok')
            setStatusText(`Connected to ${found}`)
        } else {
            setBaseUrl('')
            setClient(null)
            setStatus('down')
            setStatusText(
                baseUrlInput.trim()
                    ? 'That server is not running — check the URL and start `thoth serve`'
                    : 'Thoth is not running — start it with `thoth serve`'
            )
        }
    }, [storage, ext, baseUrlInput])

    const discard = useCallback(async () => {
        await clearDraft(storage)
        setTitle('')
        setUrl('')
        setText('')
        setReason('')
        setTags('')
        setIncludeFullPage(false)
        setResult(null)
    }, [storage])

    const save = useCallback(async () => {
        if (!client) return
        const urlTrimmed = url.trim()
        const textTrimmed = text.trim()

        if (kind === 'bookmark' || kind === 'readlater') {
            if (!urlTrimmed) return setResult({ tone: 'error', message: 'URL is required' })
            if (!isHttpUrl(urlTrimmed)) return setResult({ tone: 'error', message: 'URL must start with http(s)://' })
        }
        if ((kind === 'note' || kind === 'selection') && !textTrimmed) {
            return setResult({ tone: 'error', message: 'Text is required' })
        }
        if (kind === 'summarize' && !textTrimmed) {
            return setResult({
                tone: 'error',
                message: 'No page text to summarize — use "Ask Thoth to summarize this page" from the page menu',
            })
        }

        const draft = {
            kind,
            url: urlTrimmed,
            title: title.trim(),
            text: textTrimmed,
            category: category || DEFAULT_CATEGORY,
            folder: folder || 'inbox',
            reason: sanitizeSingleLine(reason),
            tags: parseTags(tags),
            includePageText: includeFullPage,
        }

        setSaving(true)
        setResult(null)
        try {
            // Never write a second line for an already-saved URL (#2).
            if (kind === 'bookmark' || kind === 'readlater') {
                const dup = await client.checkDuplicate(urlTrimmed)
                if (dup.exists) {
                    return setResult({ tone: 'error', message: 'Already saved — this URL is in the wiki.', path: dup.path })
                }
            }
            let res: CaptureResponse
            if (kind === 'bookmark' && includeFullPage) {
                // Full-page text only on explicit action: a knowledge note
                // with the source URL instead of a bookmark line (#7).
                const pageText = await getActivePageText(ext)
                res = await client.capture({
                    kind: 'note',
                    url: urlTrimmed,
                    title: title.trim(),
                    text: pageText,
                    folder: 'knowledge',
                    tags: draft.tags,
                })
            } else if (kind === 'summarize') {
                res = await client.summarize({ url: urlTrimmed, title: title.trim(), text: textTrimmed })
            } else {
                res = await client.capture(draftToCapture(draft))
            }
            await clearDraft(storage)
            await refreshBadge(ext, client)
            setResult({ tone: 'ok', message: `Saved to ${res.path}`, path: res.path })
        } catch (err) {
            if (err instanceof ThothError && err.status === 409) {
                setResult({ tone: 'error', message: 'Already saved — this URL is in the wiki.', path: err.path })
            } else {
                setResult({ tone: 'error', message: err instanceof Error ? err.message : 'Save failed' })
            }
        } finally {
            setSaving(false)
        }
    }, [client, kind, url, title, text, category, folder, reason, tags, includeFullPage, storage, ext])

    const visible = new Set(KIND_FIELDS[kind])
    const segmentedValue = kind === 'selection' ? 'note' : kind

    return (
        <main className="popup">
            <header className="header">
                <span className="logo" aria-hidden="true" />
                <div className="header-text">
                    <Typography.Title level={5} className="header-title">
                        Thoth Capture
                    </Typography.Title>
                    <Typography.Text className={`status ${status}`}>{statusText}</Typography.Text>
                </div>
            </header>

            <section aria-label="Server connection">
                <Space.Compact block>
                    <Input
                        value={baseUrlInput}
                        onChange={(e) => setBaseUrlInput(e.target.value)}
                        onPressEnter={() => void connect()}
                        placeholder="http://127.0.0.1:8333"
                        aria-label="Server URL"
                        spellCheck={false}
                    />
                    <Button onClick={() => void connect()}>Connect</Button>
                </Space.Compact>
                <Typography.Paragraph className="hint">
                    Auto-detects <code>thoth serve</code> (8333) then <code>make dev</code> (8334).
                </Typography.Paragraph>
            </section>

            {status === 'ok' && (
                <section className="capture" aria-label="Capture">
                    <Segmented
                        block
                        value={segmentedValue}
                        onChange={(value) => setKind(value as DraftKind)}
                        options={KIND_OPTIONS}
                        aria-label="Capture kind"
                    />

                    <div className="fields">
                        {visible.has('title') && (
                            <label className="field">
                                <span>Title</span>
                                <Input value={title} onChange={(e) => setTitle(e.target.value)} aria-label="Title" />
                            </label>
                        )}
                        {visible.has('url') && (
                            <label className="field">
                                <span>URL</span>
                                <Input
                                    value={url}
                                    onChange={(e) => setUrl(e.target.value)}
                                    readOnly={kind === 'selection'}
                                    spellCheck={false}
                                    aria-label="URL"
                                />
                            </label>
                        )}
                        {visible.has('text') && (
                            <label className="field">
                                <span>Text</span>
                                <Input.TextArea
                                    value={text}
                                    onChange={(e) => setText(e.target.value)}
                                    rows={5}
                                    aria-label="Text"
                                />
                            </label>
                        )}
                        {visible.has('reason') && (
                            <label className="field">
                                <span>Reason</span>
                                <Input value={reason} onChange={(e) => setReason(e.target.value)} aria-label="Reason" />
                            </label>
                        )}
                        {visible.has('category') && (
                            <label className="field">
                                <span>Category</span>
                                <Select
                                    value={category}
                                    onChange={setCategory}
                                    options={BOOKMARK_CATEGORIES.map((c) => ({ value: c, label: c }))}
                                    aria-label="Category"
                                />
                            </label>
                        )}
                        {visible.has('folder') && (
                            <label className="field">
                                <span>Folder</span>
                                <Select
                                    value={folder}
                                    onChange={setFolder}
                                    options={folders.map((f) => ({ value: f, label: f }))}
                                    aria-label="Folder"
                                />
                            </label>
                        )}
                        {visible.has('tags') && (
                            <label className="field">
                                <span>Tags</span>
                                <Input
                                    value={tags}
                                    onChange={(e) => setTags(e.target.value)}
                                    placeholder="comma, separated"
                                    aria-label="Tags"
                                />
                            </label>
                        )}
                        {visible.has('fullpage') && (
                            <Checkbox checked={includeFullPage} onChange={(e) => setIncludeFullPage(e.target.checked)}>
                                Include full page text (saves as a knowledge note)
                            </Checkbox>
                        )}
                    </div>

                    <Space>
                        <Button type="primary" onClick={() => void save()} loading={saving}>
                            Save
                        </Button>
                        <Button onClick={() => void discard()}>Discard</Button>
                    </Space>

                    {result && (
                        <Alert
                            className="result"
                            type={result.tone === 'ok' ? 'success' : 'error'}
                            showIcon
                            message={result.message}
                            action={
                                result.path ? (
                                    <Typography.Link
                                        href={openNoteUrl(baseUrl, result.path)}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        <LinkOutlined aria-hidden="true" /> {result.tone === 'ok' ? 'Open in Thoth' : 'Open it'}
                                    </Typography.Link>
                                ) : undefined
                            }
                        />
                    )}
                </section>
            )}
        </main>
    )
}
