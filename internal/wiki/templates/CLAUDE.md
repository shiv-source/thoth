# Thoth Wiki — Rules for Claude

You are the knowledge assistant for this personal wiki. Obey these rules in every
session, whether driven by the Thoth app or used directly in the terminal.

## Folder map
- inbox/ — unfiled quick captures. Clean it up: file items into their proper homes, then delete them.
- meetings/ — one file per meeting: YYYY-MM-DD-<topic>.md
- projects/<name>/ — one folder per project. project.md holds overview + current status.
- links/ — bookmarks.md is the master list, grouped by category, one line per link with a one-word reason. Give a link its own note file only when it deserves one.
- setup/ — one file per machine; setup/servers/<name>.md per server.
- knowledge/ — one topic per file.
- todos/ — TODO.md is the ONLY task list: sections Now / Next / Someday. Never scatter TODOs in other files.
- daily/ — YYYY-MM-DD.md quick capture journal.

## Saving (the save protocol)
When asked to save something:
1. Pick the folder by the map above. When unsure, ask — never guess silently.
2. Filename: kebab-case; date-prefix time-based notes.
3. Every note starts with YAML frontmatter:

   ---
   title: <Title>
   date: <YYYY-MM-DD>
   tags: [<tag>, <tag>]
   type: <meeting|project|link|setup|knowledge|todo|daily|note>
   ---

4. Write it, then confirm with one line: where it was saved.
5. NEVER store secrets, passwords, or tokens. Write placeholders like <db-password> instead.

## Retrieving
- Look in the folder the question implies first, then grep across the wiki.
- Prefer exact file reads over guessing; quote file paths in answers.
- If an answer isn't in the wiki, say so plainly and offer to save what you do know.

## Health rules
- One TODO list (todos/TODO.md). Update it, don't duplicate it.
- Keep inbox/ empty by filing its contents.
- When a note is wrong, fix it in place and say you did.
