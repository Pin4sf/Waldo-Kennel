# Backend takeover review: aff83c3a0

Reviewed committed checkpoint aff83c3a0deafbf984602645b711d95c1171c625, based on 670a42238. Takeover branch: codex/kennel-backend-takeover. Claude's clean worktree was preserved. Nothing pushed or merged.

## Verdict

Useful integration and meaningful automated evidence, but not approved for merge or full Mission integration. R1 proof read ordering and R3 argument preservation are corrected. R2 now reserves executions and persists observations, fixing repeated-tick reruns, but still mishandles an active reservation. Four additional regression assertions below failed deterministically at the real service entry points using existing test adapters. These are not live-provider tests.

## Reproduced findings

1. **P1: changed run commands reuse the same key without conflict.** service/outcome/run_intent.go:78 checks only Outcome identity on replay. Start followed by Cancel with the same key returns success but remains running. Persist and compare a complete request fingerprint, including command, reviewed Plan/Contract and concurrency expectation; apply it inside the store transaction as well as the service replay path. Test changed command, Plan and cross-Outcome reuse, including concurrent writers.

2. **P1: Start ignores ExpectedContractRevision.** run_intent.go:109 copies the field and then replaces it with the latest Plan binding without validating the owner's supplied revision. The regression passes expected revision 999 against Contract 1 and Start succeeds. Validate the exact reviewed authority for start/resume; retain usable stop controls when authority is stale. Revalidate authority at durable write/admission boundaries.

3. **P1: document approval accepts an unnamed snapshot.** documents.go:102 skips its comparison when expectedDigest is blank, despite the documented requirement. A selected document is approved with an empty string. Require the digest and validate the current selection at the write boundary; return a typed refusal, not a silent approval.

4. **P1: a live check is classified as interrupted.** checks.go:127 assumes any existing reserved run belongs to a dead process. A second reconciliation while the original runner is executing changes the reservation to unknown, invalidating its later observation. The regression re-enters the service while the runner callback is active and observes exactly this transition. Track actual invocation ownership/recovery epoch; only a proven abandoned invocation becomes unknown. A reserved row alone is not evidence of process death. Keep once-only invocation and immutable proof reconstruction.

## Additional source-traced concurrency concern

ExpectedGeneration is checked outside AppendRunIntent's transaction; the store does not receive an expected generation and simply appends current+1. Two commands based on the same observed generation can therefore both commit. Also, StartAttempt checks pause/cancel near line 179 but creates its Attempt/fence near line 272 without including the run generation in the atomic admission request. Pause can land between those operations. Repair both transaction boundaries and add deterministic service/SQLite interleaving tests before calling pause/replay restart-safe. An in-memory-only lock is insufficient for the durable guarantee.

## Evidence

Existing focused packages passed: service/outcome, service/intelligence, daemon, session_manager, storage/sqlite/store, artifactstore. Log: /tmp/kennel-aff83c3a0-review-tests.log.

Four new review assertions failed: /tmp/kennel-aff83c3a0-review-regressions.log. Reproduction source is stored beside this note as kennel-aff83c3a0-review-regressions.go.txt; copy to backend/internal/service/outcome/takeover_review_test.go and run go test ./internal/service/outcome -run TestTakeoverReview -count=1 from backend. It is saved outside the suite until the repair slice promotes it into permanent passing tests.

No full suite/race rerun, real provider, booted-daemon journey or package acceptance was performed by this review. Claude's wider verification remains reported evidence, not independently repeated here.

## Takeover order

1. Fix the reproduced approval/replay/active-check defects and atomic run-intent/admission boundary. Preserve the original checkpoint and reviewable commits.
2. Re-review full cumulative backend contracts, including approved checks, run intent, supplied documents and the matching UI. The isolated interface commits alone do not deliver implemented run/document behavior; integration must use a reviewed cumulative base.
3. Complete typed setup errors and replan replay, rework, durable delivery, and Outcome-scoped agent interaction/bridge. The original four-item handoff omitted the explicit durable Outcome conversation requirement; do not substitute Project chat.
4. Finish usage attribution, bootstrap cleanup and provider/plugin setup. Benchmark optional model delegation later.
5. Integrate the UI in a separate reviewed worktree and run the real agent-led Outcome journey, then packaged acceptance.

No redesign or second orchestrator is required. The goal remains one user-led responsibility with agent-led planning/execution and daemon-enforced authority.
