# scripts/

Repo helper scripts — run manually by developers and agents, not part of CI.
(CI lives in `.github/workflows/`; commit hooks in `.husky/`.)

- `graph-check.sh` — staleness guard for the committed graphify graph: exits 1
  when a tracked source (Go, TS/TSX, md, yaml) is newer than
  `graphify-out/graph.json`; run `graphify update .` and commit the refresh.

Changes here carry the `tooling` area label.
