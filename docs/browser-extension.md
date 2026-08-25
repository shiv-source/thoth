# Browser extension

Thoth Capture is a Manifest V3 browser extension for Chrome and Firefox that makes capturing one click from anywhere: **bookmarks, text selections, and quick notes** land in the same wiki — through the same rulebook save paths — as everything the dashboard and the assistant write.

There is one capture surface and one save path. The extension never reimplements wiki rules; it posts to the unified `POST /api/v1/capture` endpoints, so a capture can't drift from the wiki contract.

## Install (load unpacked)

Build the packages first:

```sh
make ext-build
```

- **Chrome / Edge** — `chrome://extensions` → enable *Developer mode* → *Load unpacked* → choose `browser-ext/dist/chrome`
- **Firefox** — `about:debugging#/runtime/this-firefox` → *Load Temporary Add-on* → choose `browser-ext/dist/firefox/manifest.json`

## How it works

The extension discovers the server by probing `http://127.0.0.1:8333` (`thoth serve`) then `:8334` (`make dev`). A custom base URL (set in the popup) overrides discovery. The server is auth-less localhost, and MV3 `host_permissions` let the extension `fetch` it directly — **no server CORS changes needed**.

A custom URL on `127.0.0.1`/`localhost` needs nothing extra. A custom server on any other host requests a one-time `optional_host_permissions` grant the first time you connect (the default install stays localhost-only — nothing broad is requested up front). If a custom URL is unreachable, the popup says so instead of silently reconnecting to the default port — it never lies about where captures land.

### Context menus

Right-click on any page:

- **Bookmark page to Thoth** — metadata only (title/URL/reason), appended to `links/bookmarks.md` grouped by category
- **Save selection to Thoth** — the quote saved as a note with the page URL in `source:` frontmatter; the note title is derived from the quote's first sentence (not the page's generic title) and the capture is auto-tagged with the source domain, so every capture from a site stays grouped
- **Add to Thoth read later** — appended to the `links/read-later.md` queue
- **Ask Thoth to summarize this page** — the assistant summarizes the page text into a `knowledge/` note with `source:` and the source-domain tag

A click stores a **draft** and opens the popup prefilled for review — the draft step keeps the wiki free of inbox garbage. If the URL is already saved, the popup shows **"Already saved → open it"** instead of writing a duplicate.

### The popup

- **Server status** — connected / "Thoth is not running — start it with `thoth serve`"
- **Kind tabs** — Note, Bookmark, Read later, Summarize
- **Editable fields** — title, URL, text, reason, category, folder, tags
- **Bookmark category** — the popup remembers the last category you used and defaults to it (falling back to `unfiled` on first use); it never guesses from the URL, and it's always editable before saving
- **Include full page text** — off by default; ticking it captures the whole page as a `knowledge/` note (with the source-domain tag) instead of a bookmark line

After any save, an **Open in Thoth** link opens the dashboard at the saved note.

### Triage in the dashboard

Read-later is a queue, not a graveyard: the dashboard's **Read later** card (Overview) lists `links/read-later.md` and lets you open each link, promote it to a bookmark, or clear it — so a "read later" capture becomes "read, decided, filed" without re-capturing.

### Toolbar badge

The toolbar badge shows how many unfiled captures are sitting in `inbox/` (from `GET /api/v1/capture/inbox-count`). It refreshes on install, startup, every capture, and on a 5-minute alarm.

## Privacy and scope

Capture is **default-safe**: a bookmark is title/URL/reason only, and full-page text is captured only when you explicitly ask for it (the summarize menu or the "include full page text" toggle). Nothing leaves your machine — the extension talks to your local server, and that server is localhost-only.

## Development

`browser-ext/` is a pnpm workspace member built with the same stack as the dashboard — **React + antd** for the popup, plain TypeScript for the framework-free core.

```sh
pnpm --filter thoth-ext test        # vitest (core logic + popup components)
pnpm --filter thoth-ext typecheck   # tsc -b
pnpm --filter thoth-ext lint        # eslint
pnpm --filter thoth-ext build       # esbuild → dist/{chrome,firefox}
pnpm --filter thoth-ext e2e         # Playwright: load the built extension, drive the popup
make ext-check                      # the four gates
make ext-e2e                        # Playwright e2e (browsers: pnpm exec playwright install chromium)
```

Shared logic lives in `browser-ext/src/core` and is dependency-injected (`BrowserAPI` + `StorageLike` seams) so tests run under Node with fakes; `src/chrome` and `src/firefox` are thin per-browser manifests and entries. Full component map: [Components](components.md).
