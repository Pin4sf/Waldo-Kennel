# Contract editing and permission-aware Plan drafts

Local launch candidate follow-up to b5a9b281c. Nothing pushed or deployed.

## Changes

- Mission Contract tab now offers Edit Contract after confirmation. Goal, criteria, per-criterion evidence expectations, review, constraints, non-goals, pause conditions and permissions can be revised. Saves append an immutable revision using expectedRevision. Temporal conditions, facets, clarification and execution preference are retained; title editing is not included.
- A newer Contract arriving through CDC preserves an open draft and disables saving it until the owner discards/reopens against current facts. Revision does not stop an already running session or reauthorize an old Plan.
- Revision API adds criterionEvidence by criterion position, temporalCondition and facets. The daemon binds expectations to freshly assigned criterion IDs and rejects positional count mismatch before writing.
- Plan input now contains the actual frozen authority flags. For Contracts forbidding command execution, its schema requires null checkCommands and only non-executing intents. This is proposal guidance; deterministic Plan validation remains authoritative.
- The first live retry proved prompt instructions alone were insufficient: Nano still proposed checks without execution permission. The conditional schema addresses that observed failure without granting permissions or weakening checks.

## Verification

- Intelligence, Outcome service and HTTP/spec tests pass. API source and generated contracts regenerated together. Frontend typecheck and packaged build pass.
- Frontend full run: 225 of 226 files passed, 2737 tests passed, 6 skipped. One localization test caught two hardcoded accessibility labels; those were fixed and the localization/editor tests then passed (2 files, 3 tests). This is not a claim of a single clean full-suite invocation.
- An earlier unprivileged frontend run failed listener tests because the sandbox denied loopback access; the above full run had the required access.
- Lint reports 15 findings; none in this slice's changed files. The log is /private/tmp/launch-contract-editor-lint.log.
- Native Electron: opened the real agent-orchestrator Outcome, edited its pause condition, saved revision 2, restarted the isolated app/daemon and observed revision 2 again. Daemon read confirms two evidence expectations bound to the new criterion IDs and the unchanged read-only permission ceiling.
- New UI text is routed through locale keys; non-English catalogs currently use English copy for this slice.

- Final native Plan retry PASSED with the existing gpt-5-nano configuration: Contract 2 / Plan 1 proposed, one Codex WorkUnit, only worktree.read, no deterministic checks. Graph preview visible in Plan tab. Left awaiting owner authorization. This proves this read-only drafting case, not all model outputs or execution.

## Remaining boundaries

No Outcome execution, proof, delivery or acceptance is established by the editor test. A user's broader OpenAI key has not yet been tested in this run. Settings accept an explicit model ID and optional effort; only models compatible with the structured Responses path are candidates, not every model returned by an account catalog. Codex reasoning confinement/identity binding, Claude resume conformance and OpenCode live conformance remain separate launch work.
