# A2 deferred scope — supplied-document context and non-repository custody

Date: 2026-09-09. Status: **specified, not implemented.** Deferred by explicit
owner decision so Phases B, C and D land first; the owner asked for the working
software loop in hand before this capability is built.

This is not a descoping. A2 remains part of the Kennel Work completion
assignment and is picked up after the B–D checkpoint. This document exists so
that when it resumes, nothing has to be rediscovered and nothing already built
has to be unpicked.

## Why it was safe to defer

A2-2 adds a new *source* of grounding context. Phases B, C and D operate on
Contract, Plan, WorkUnit and Attempt regardless of where context came from, and
the repository context path already works — so the input path is never empty,
only narrower.

## What A1/A2 already established, and does not need redoing

| ID | Requirement | State |
|---|---|---|
| A2-1 | Grounding is bounded, secret-safe, symlink-confined, cancellable and fails closed on a Git error | **verified independently** (not accepted on report) |

A2-1's evidence: `TestBuildRepositoryContextBoundsFilesAndExcludesIgnoredSymlinkedSecrets`
writes real `ignored-secret-canary` / `unignored-secret-canary` files, a symlink
escaping the root, and a real `git init`, then asserts none of that content
reaches the snapshot. `TestIgnoredByGitDistinguishesNotIgnoredFromGitFailure`
proves a Git failure is not read as "not ignored", and the call site at
`service/intelligence/repository_context.go:130` propagates that error out of
the `WalkDir` callback — so a Git failure aborts the context build rather than
admitting the file. All four grounding tests pass on this branch.

## Remaining A2 work

| ID | Requirement | Existing source | State |
|---|---|---|---|
| A2-2 | Supplied-document context with an explicit custody model | repository-only | **missing** |
| A2-3 | Bounded context enters the **Plan** input digest as well as the Contract's | Contract digest confirmed (`d617b4ee3`) | **partial** |
| A2-4 | Unresolved assumptions/blockers surfaced before approval | migration 0117 stores them; Plan API returns them | **partial** |

### A2-2 design constraints

- Reuse the existing attachment/input staging the daemon already performs for
  chat attachments. Do not build a general ingestion platform, and do not add an
  indexing or vector store.
- Define the supported local text/document formats explicitly and return
  truthful errors for the rest. An unsupported input is *unavailable*, not an
  implied capability, and must be refused **before** authorization rather than
  failing mid-execution.
- Never silently `git init` a supplied folder, and never present a plain folder
  as having worktree isolation.
- Paths and content from supplied documents are untrusted data, exactly as
  repository content is.
- A request needing external research under a no-network policy explains the
  missing capability. It never fabricates citations.

## Consequences carried into B, C and D

These were accepted as conditions of deferring, and are the reason this
deferral does not create a retrofit:

1. **Phase B shows no supplied-document context control.** A control that
   cannot work is a dead interaction; AGENTS.md requires hiding or disabling an
   untruthful control rather than leaving it visible. The repository context
   path is offered; the document path is absent, not broken.
2. **Phase C designs custody for a staged workspace generally**, with the Git
   worktree as one implementation rather than the only one. Custody and artifact
   retention are built in C, so if C assumed Git alone, A2-2 would later need a
   second custody path retrofitted into a frozen receipt model — the same
   retrofit trap already closed for export provenance. Building C against a
   staged-workspace abstraction is what keeps A2-2 additive.
3. **Phase D's document row is recorded blocked, not passed.** D's exit criteria
   name "a supported document task uses appropriate artifact/criterion checks
   without meaningless software test commands". The check model is built
   criterion- and artifact-based rather than test-command-based, which is the
   correct design regardless — but the row cannot be *demonstrated* until A2-2
   lands, so it stays open.
4. **Launch claims stay narrower than the domain model.** Until A2-2 lands, only
   the repository/software Outcome path is tested and therefore only that path
   may be advertised. Acceptance matrix rows 3 and 15 remain open.

## Resumption checklist

1. Re-read this document and the A2 rows in
   [the completion ledger](../../verification/kennel-work-completion-ledger.md).
2. Confirm Phase C's custody/artifact model still exposes a non-Git staged
   workspace path; if C drifted to Git-only, fix that before adding intake.
3. Implement A2-2 intake + custody, then A2-3 and A2-4.
4. Run the A2 negative tests the assignment requires: supplied-document canary
   exclusion, unsupported-format refusal before launch, changed context changes
   both the Contract and Plan digests, and no pre-approval Attempt.
5. Only then extend the advertised Outcome kinds and reopen matrix rows 3 and 15.
