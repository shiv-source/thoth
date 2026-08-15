❯ What are the additional features we can provide ?

  Thought for 16s

Great moment to ask — the foundation now supports a lot. Here's a curated menu, grouped and ranked by value/effort:

Capture & organization (quick wins)
1. Quick-capture composer — a small "add note" form in the sidebar (title + content + type) that writes directly via a new PUT /api/notes — capture without chatting
2. Inbox triage button — one click asks Claude to file everything in inbox/ per the rulebook, then shows what moved
3. TODO dashboard — a Tasks tab rendering todos/TODO.md's Now/Next/Someday with check-off (small PATCH to toggle lines)

Knowledge & retrieval
4. Clickable tags — tags in search results become filters; tag cloud in the sidebar
5. Related notes — "see also" suggestions per note (computed via Claude on save, stored in frontmatter — no embeddings needed)
6. In-browser note editing — full editor with live preview (bigger: write path, conflict handling)
7. Wikilinks + backlinks — [[note]] links with a backlinks panel

Automation & peace of mind
8. Auto git sync — commit+push the wiki on an interval or after each save (the git plumbing from the settings tab already exists)
9. Daily/weekly digest — Claude summarizes your daily/ + meetings/ on demand ("summarize this week")
10. Meeting capture template — structured "New meeting" form (attendees, decisions, actions) auto-filed per the rulebook

Polish (previously deferred)
11. ⌘K command palette — search any action/note instantly
12. Keyboard shortcuts — ⌘↵ send, Esc close, arrows navigate
13. Manual dark/light toggle — in the settings modal

My picks for the next round: 2 + 3 + 8 + 11 — they compound (capture → organization → automation → navigation) and all reuse existing plumbing. Which ones interest you?
──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────