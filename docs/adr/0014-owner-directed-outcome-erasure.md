# ADR 0014: Owner-directed Outcome Trash and permanent erasure

Status: Implemented for review, 2026-09-11.

The owner needs to discard test Outcomes and recreate them. Execution history remains immutable during normal work; explicit owner-directed permanent deletion is a separate lifecycle boundary, not a rewrite of proof or acceptance.

## Decision

- Move to Trash hides the selected responsibility tree from active reads and scheduling. Restore preserves its Contract and history and does not authorize or start execution.
- A preview enumerates related records, session references and workspace paths. Active/unconfirmed Attempts, unreleased custody, running authorization, live sessions, pending deliveries and shared references block deletion.
- Permanent deletion requires the current Contract revision and the explicit confirmation word `confirm`; the dialog displays the Outcome title. It first enters durable `erasing` state in Trash. Restore is then unavailable because cleanup may have removed retained bytes. Failures leave this state and the records inspectable for explicit retry.
- Session/workspace teardown uses the existing session manager. Unknown provider state and dirty workspaces are not force-cleaned. Owned workspace absence is checked before removing retained snapshots and again before the final database transaction.
- The store traverses an explicit table allowlist and database relationships. Project conversations and other shared references are blockers, not incidental ownership. Imported projects, export destinations, provider-side history, logs and backups are outside erasure.
- Immutable DELETE guards permit only the exact row IDs populated by the private purge transaction. The scope is cleared before commit; failures roll back the scope and record deletion together. Normal history edits/deletes retain their guards. A database CDC trigger emits the removed Outcome identity.
- Trash admission guards fence late Attempt, Plan, Contract, run-intent, document and delivery insertion, and session resurrection. New schema tables do not automatically acquire deletion authority.

## Limits

This is application-level erasure, not forensic secure deletion. It does not erase remote provider history, exports, backups or logs. A cleanup failure requires an explicit retry from Trash; it is not reported as success. The existing ambiguous launch Attempt still requires stop reconciliation before deletion can proceed.
