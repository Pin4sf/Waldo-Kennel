# CLAUDE.md

Read and follow [`AGENTS.md`](AGENTS.md) for repository layout, commands, coding conventions, and hard rules.

## App state lives under `~/.kennel` only

All app state, the daemon's data dir, `running.json`, worktrees, and the Electron
supervisor's `userData` (Chromium cache, cookies, local/session storage, crash
dumps), must resolve under `~/.kennel` (overridable via `KENNEL_DATA_DIR`/`KENNEL_RUN_FILE`).
Never write to or read from `~/Library/Application Support` or any other OS-default
app-data location. `frontend/src/main.ts` pins Electron's `userData` to
`~/.kennel/electron`; do not remove that override. See the hard rule in `AGENTS.md`.

## Design System

Always read [`DESIGN.md`](DESIGN.md) before making any visual or UI decision —
**start with the "Kennel orchestrator system, Figma" banner at the top**, which
governs the current look.

The app follows the **Kennel orchestrator design system** authored in Figma
(file `Dl0WP9uIvx6QbSzZi7cZQY`, section `2984-17556`), per explicit user decision
2026-08-22. This **supersedes the earlier "clone agent-orchestrator verbatim"
direction**, which is kept in DESIGN.md for provenance only. Kennel Island is
the same system tuned for the notch: reduced text hierarchy, simpler composition,
and a `#000000` background at all times so it reads as continuous with the physical
camera housing.

Everything visual resolves through `frontend/src/styles/tokens.css`; never hardcode a
hex, radius, or size in a component. Build new UI from shadcn primitives
(`components/ui/*`) where a component fits. Do not deviate without explicit user
approval. In QA/review, flag renderer code that diverges from the Figma system — do
**not** re-flag agent-orchestrator or old design-reference mismatches.

When showing or demoing frontend changes, run `kennel preview [url]` from inside the
session so the change renders in the desktop browser panel (the inspector rail's
Browser tab); do not just describe it.
