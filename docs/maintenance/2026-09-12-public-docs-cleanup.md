# Public documentation cleanup — 2026-09-12

Base: `b9f69d949` (`origin/beta`). At review, `origin/main` had the same tree;
its three additional commits were promotion merges. This local change targets
`beta`; it does not update either remote branch or publish a release.

## Removed

Each file below explicitly declared itself superseded/retired and had no inbound
filename reference in repository Markdown before removal. They were pointer
stubs, not unique implementation or verification records. Originals remain in
Git history.

- `docs/HANDOFF.md` — retired handoff or delivery pointer.
- `docs/product/kennel-v1-excalidraw-session-seed.md` — retired design-session seed.
- `docs/superpowers/plans/2026-08-24-work-lane-execution-handoff.md` — retired handoff or delivery pointer.
- `docs/superpowers/plans/2026-08-30-composed-outcomes-handoff.md` — retired handoff or delivery pointer.
- `docs/superpowers/plans/2026-08-29-composed-outcomes-program.md` — retired handoff or delivery pointer.
- `docs/superpowers/plans/2026-08-25-work-control-plane-delivery.md` — retired handoff or delivery pointer.

## Retained and consolidated

Ten explicitly superseded Markdown stubs were inspected: six disconnected
stubs were removed; four with inbound historical/research references remain.
ADRs, current authority, active roadmap specifications, source provenance and
verification records remain. Age alone was not a deletion criterion. The
remaining dated documents are scoped records or future research, not a claim
that every historical instruction is current.

README and the documentation map now point to AGENTS.md's current authority
order. STATUS keeps one current inventory and links existing evidence records;
its contradictory dated branch inventories and temporary local log listings
remain recoverable from Git history rather than a duplicate status archive.
The stale claim that the frontend submits Project provider preference was
removed after checking the current Outcome run hook.

Validation: local Markdown target existence for changed entry documents,
repository reference scan for removed filenames, and `git diff --check`.
No runtime code changed and no new provider or packaged acceptance is claimed.

## Maintainer gaps (read-only check)

The repository is already public. At this review GitHub Actions was enabled;
that does not prove required CI is passing. `beta` had no classic branch
protection, `main` had no required status checks and zero required approving
reviews, and the rulesets API returned no rulesets. Private vulnerability
reporting was disabled and the releases API returned no releases. These settings
were not changed; track maintainer work through issue #118 and the roadmap.
