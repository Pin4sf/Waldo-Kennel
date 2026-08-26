# Implementation issues and follow-ups

Last updated: 2026-08-26 during issue #32 verification.

This file records verified follow-ups discovered while implementing a feature. Independent work that is large enough to schedule and review separately must also have a dedicated GitHub issue; this file does not replace the canonical issue tracker.

## Escalated: repository-wide lint baseline

- GitHub issue: [#75 Restore the clean repo-wide lint baseline](https://github.com/Pin4sf/Waldo-Kennel/issues/75)
- Status: open; not caused by issue #32.
- Evidence: `npm run lint` exits 1 with 20 findings on both the issue #32 branch and a clean checkout of `origin/beta` at `296b92a4d822c221ed586a24e13cfa110381419c`.
- Breakdown: 1 errorlint, 1 goimports, 1 prealloc, 16 revive, and 1 staticcheck finding.
- Boundary: fix this in its own coordinated lane. Do not absorb unrelated Home, Outcome, agent-role, or DeepSeek cleanup into issue #32.

## Existing integration follow-ups

These observations are already covered by [#40 Integrate Work with Home through shared intake and ResponsibilityLink](https://github.com/Pin4sf/Waldo-Kennel/issues/40), so they do not need duplicate GitHub issues:

- An unconfirmed `IntakeSession` is durable and readable after a daemon restart, but the Work board does not yet discover and resume it automatically.
- `ResponsibilityLink` accepts the Home source responsibility as an opaque identifier. Resolving and presenting the canonical Home-side responsibility belongs to the separate Work/Home consumption layer.

## Closed by issue #32

- Offline capture failure now keeps the entered statement visible and explicitly marks it unsaved.
- Simple Outcomes advance without a clarification; adaptive intake asks no more than one material question.
- Confirmation and ResponsibilityLink creation are idempotent, stale revisions are rejected, provenance stores references instead of transcript copies, and CDC remains trigger-backed.
