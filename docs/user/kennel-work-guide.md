# Kennel Work guide

Kennel Work manages Outcomes. A provider session is only the execution
machinery used to make an Outcome true; it is not the responsibility itself.

## Start a software Outcome

1. Open Work and choose a registered Project.
2. Describe the Outcome and its success criteria. Keep criteria observable.
3. Review the proposed Contract and Plan. Check the WorkUnit order, required
   capabilities, stop conditions, and verification expectations.
4. Approve the Plan. Approval authorizes the exact provider/model binding; it
   does not start execution.
5. Choose Start in Act & Observe. The graph is serial in the current launch
   build. A WorkUnit waiting for proof or custody is not an error.
6. Pause, cancel, or answer an Outcome-local question from the Work surface.
   A quiet or unreachable provider is not proof that work stopped; recovery
   remains conservative and may require your confirmation.
7. Review the retained result and deterministic checks in Prove & Close.
   Provider completion, a commit, or a green check does not accept an Outcome.
8. Explicitly accept the Outcome, request rework, or reopen it. Only the owner
   creates the AcceptanceDecision.

## Reasoning and provider setup

Waldo reasoning credentials are separate from provider authentication. A
configured key is not the same as a verified key. Use Settings → Reasoning →
Verify after changing provider, model, endpoint, or credential. Verification
is bound to that exact configuration and old in-flight probes cannot stamp a
replacement configuration. Kennel provider readiness is machine- and
capability-derived; there is no silent fallback to another provider.

## Results and restart recovery

Ended Attempts are retained before they can be classified as successful.
Retained bytes include tracked committed changes since the recorded base,
dirty/staged edits, untracked files, deletions, executable mode, and binaries.
Secret-like files, symlinks, special files, inconsistent reads, and bounds are
reported as unsupported or incomplete rather than silently omitted.

Restarting the daemon replays durable liveness, retention, receipt, proof and
custody facts. It does not repeat a provider call merely because a process was
quiet. Never delete a workspace or artifact directory that Kennel has not
identified as owned and safely retained.

## Supplied documents and delivery

The current launch build advertises the repository/software path. Supplied
document Outcomes remain a bounded staged-folder capability under active
implementation; PDF/OCR, external research, and network ingestion are not
implied. Do not treat a plain folder as a Git worktree.

When delivery is available, Export is an owner-triggered copy of an accepted
result (or visibly labelled draft where allowed). It preserves additions,
deletions, binary bytes and modes, checks the recorded base, refuses unsafe
destinations and does not merge, deploy, or delete the source workspace.

For local dogfood, set `VITE_KENNEL_WORK_LAUNCH=1` in the renderer environment
to make Work the focused shell and hide ambient Waldo chrome. This is
reversible and does not alter durable data or historical deep links.
