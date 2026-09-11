# Workspace spawn failure and expanded graph

Base: d0b7c2947, isolated launch worktree. No push or deployment.

## Observed failure

The real agent-orchestrator Attempt att-f4c4301d-6e2b-4948-8bd4-1bf7ef5b7d52 recorded admission_ambiguous because workspace creation refused branch kennel/agent-orchestrator-1/root. That branch is already checked out under the separate 20260907T082847Z audit profile. The current profile had no session rows. The recorded error occurred before provider launch; it is not evidence of a Codex authentication or stream-binding failure.

Custom data directories previously used the same generated branch namespace as the default profile despite independent session counters. They now derive a stable hashed namespace from their absolute data directory. Default production, default dev and explicit user branch behavior remain unchanged. Existing branch/worktree metadata is not rewritten; the old worktree is untouched.

The session manager now marks workspace-creation failure as a known pre-launch boundary. Outcome admission records admission_failed, ends the Attempt and permits deliberate retry instead of calling it an unknown activation. This does not classify provider-start failures or alter unknown-run fencing.

The historical stuck Attempt is NOT retroactively changed based on error-string parsing. Its recovery API requires explicit owner confirmation of a stopped provider. Confirmation was requested; no recovery assertion or duplicate Attempt was submitted in this slice.

## Graph

Plan and Execution graph/table controls now include Expand graph, using the existing accessible dialog. It presents the same units and selection in a 95vw by 90vh view, with zoom and selected-node details. Native Electron opening and full-window layout were visually verified.

Testing exposed a misleading graph label: the schedule calls a queued/unconfirmed Attempt executing because it holds custody. The graph now labels that state "Attempt open" and directs the owner to its session/recovery status, rather than claiming provider activity. The Attempt card continues to show start unknown.

## Checks

- Red: actual session-manager test reproduced identical generated branches across two independent profiles.
- Green: session_manager, service/outcome and daemon tests pass; changed manager and Outcome packages pass race checks; vet passes on manager, Outcome and daemon.
- Service test proves known workspace failure becomes terminal and a retry is admitted in the fake-store harness. It is not a new independent SQLite custody-release test.
- Frontend typecheck, package build and graph/localization tests pass (3 files, 18 tests).
- No successful native Codex Outcome execution or session attachment is claimed yet. Live retry remains dependent on historical Attempt recovery confirmation.
- Codex-backed planning/reasoning remains a distinct unfinished adapter/confinement/account-binding gate.
