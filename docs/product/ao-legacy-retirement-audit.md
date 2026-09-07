# AO legacy retirement audit

- **Status:** Approved retirement architecture; destructive file removal remains an issue-sized implementation step
- **Audit date:** 2026-08-21
- **Scope:** Agent Orchestrator names, donor applications, packages, documentation, assets, compatibility seams, and provenance required for the Waldo Kennel build
- **Replacement:** Waldo product identity with Kennel as the desktop execution presence
- **Plan:** [AO legacy retirement implementation plan](../superpowers/plans/2026-08-21-ao-legacy-retirement.md)

## Decision

AO history is evidence and source provenance, not current product identity. Cleanup therefore uses four dispositions:

- **Retain:** attribution, historical migrations, and active compatibility seams whose removal would break data or reviewed upstream ports.
- **Rename now:** active user-facing, diagnostic, generated, and visual product identity.
- **Retire/remove:** disconnected donor products, stale plans, packages, translations, screenshots, and build jobs after their consumers are removed.
- **Defer behind a measured migration:** Go module/source entrypoint and project-local `.ao` file-format names.

Zero occurrences of `AO` is not the goal. Zero misleading active product identity is the goal.

## Observed impact surface

| Surface | Observed state | Disposition |
| --- | --- | --- |
| `NOTICE`, `LICENSE`, `upstream.json`, `docs/upstream-provenance.md`, `scripts/compare-upstream.sh` | Required Apache-2.0 attribution and reproducible source provenance | Retain permanently. |
| `backend/cmd/kennel`, Go module `github.com/aoagents/agent-orchestrator/backend`, generated sqlc imports | Thousands of source imports and the reviewed upstream synchronization seam; packaged binary is already `kennel` | Retain for the first product milestones; migrate only through a separate mechanical module-path issue after downstream inventory. |
| Historical SQLite migrations, persisted provider/session identities, legacy Git author/commit recognition | Required replay and backward compatibility | Retain; never rewrite merged migrations or historical values. New presentation uses Kennel/Waldo. |
| Project-local `.kennel/attachments` and `.kennel/launch.json` | Active inherited project file formats | Retain until an additive `.kennel` format, dual-read migration, and downstream inventory exist. |
| Active Electron/backend UI strings and diagnostics | AO appears in startup, daemon, browser, project, switching, feedback, prompts, and approximately 192 locale entries | Rename to Kennel or Waldo according to ownership; preserve only explicit historical/import wording. |
| `frontend/assets/ao-logo.svg`, packaged `icon.*`, tray icons, renderer AO logo | Active desktop visuals still use the AO elephant/wand | Replace from canonical Waldo paw source, then remove AO visual assets after packaged-identity and visual checks. |
| `frontend/src/landing` | 427 tracked files, 51,108,774 bytes; AO marketing/docs product; only foundation bootstrap/test/audit and Dependabot consume it | Remove as a product donor and remove its build/dependency jobs. Waldo's public site remains a separate repository. |
| `packages/mobile` | 233 tracked files, 2,270,132 bytes; AO remote-supervisor app, not the Health-aware Waldo mobile architecture | Remove after extracting only source-independent protocol lessons into docs. Do not rename it into the future mobile app. |
| `packages/ao` and four platform packages | Six frozen npm compatibility files; explicitly forbidden from publication | Remove after confirming no package/workflow dependency and preserve deprecation history in this audit. |
| `docs/ao-start-bootstrapper-and-npm-deprecation.md` | Obsolete AO distribution implementation specification conflicting with Kennel's desktop release contract | Remove; Git history and this audit preserve its disposition. |
| `translations/README.*` | Seven stale translations of the AO README | Remove. Reintroduce localization only from current Kennel source copy and tests. |
| root `assets/*` and `docs/assets/readme/*` AO marketing screenshots | 24 files totaling about 8.7 MB; almost all consumers are stale translated readmes or mock filenames | Remove unreferenced files with the donor surfaces; keep only files proven by an active test/runtime consumer. |
| `docs/architecture.md`, `docs/stack.md`, `docs/harnesses/*`, telemetry/cloud documents | Mixed active chassis facts and stale AO product/cloud wording | Rewrite current chassis docs to Kennel vocabulary; archive or remove cloud/product documents that have no current Kennel consumer. Do not rewrite historical ADR decisions. |
| `packages/cloud-client`, LAN/mobile bridge code, release/pod helpers | Mixed: some tested shared code, some donor-only behavior | Inspect per consumer. Retain a component only when current desktop/backend build or an accepted plan names it; otherwise retire in a separate small PR. |

## Canonical replacement asset

Use the Waldo paw mark at `/Users/shivanshfulper/Developer/Pin4sf/waldo-landing/components/assets/waldo-logo.svg`, source repository commit `962d022aeab2e100cd1d655fcaeb6edbf6a062f6`, SHA-256 `40071ff75328a5cf74c504ee7fe8e53cf50ce8d858046720305f6576c910eef2`.

Copy it into Kennel as a reviewed source asset; do not add a runtime dependency on the landing repository. Generate checked-in desktop formats from that source and verify light/dark, small-size, packaged, tray, and accessibility behavior. The product label remains `Kennel`; the mark identifies the one Waldo agent present through Kennel.

## Consumer and migration sequence

1. Add exact source/asset checks and replace active AO product strings and visuals.
2. Remove landing/mobile/AO-package jobs from bootstrap, foundation, audit, and Dependabot while preserving current desktop/backend coverage.
3. Delete the disconnected donor trees, stale translations, obsolete AO plan, and now-unreferenced media in one reviewable retirement PR.
4. Rewrite current chassis documentation and add an automated active-surface AO vocabulary denylist with explicit compatibility/provenance allowlist.
5. Evaluate cloud-client and helper consumers separately; do not delete shared code merely because it originated in AO.
6. Defer Go module and project-local format migration until their own expand–migrate–contract specifications exist.

## Rollback and falsifiers

Before deletion, tag the last donor-bearing commit in the issue record and record exact path lists. Git history is the recovery mechanism; no runtime data is deleted.

Stop a retirement slice when:

- a current desktop/backend build imports the proposed target;
- removing it reduces an accepted test boundary without replacement;
- the path contains attribution required by `NOTICE` or Apache-2.0;
- persisted state, project files, or historical sessions would become unreadable;
- packaged assets cannot be replaced and verified in the same PR;
- a downstream repository or release workflow is an unknown active consumer.

## Verification contract

- `rg` finds AO product wording only in the explicit provenance/compatibility allowlist.
- `npm run bootstrap`, `npm run test:foundation`, `npm run audit:production`, `npm run lint`, frontend tests/typecheck/build, and packaged identity pass without donor jobs.
- `npm --prefix frontend run package:identity` and macOS artifact verification pass with Waldo/Kennel assets.
- no `packages/ao*` publish path, AO release target, AO global state fallback, or AO updater identifier remains.
- current SQLite histories and historical provider identities still open.
- `scripts/compare-upstream.sh` and the provenance record remain usable.

This audit authorizes planning and issue creation. The implementation issue must list the exact deleted paths and obtain confirmation before its destructive removal commit.
