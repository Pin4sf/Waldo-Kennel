# PR #110 launch-fix handoff

Date: 2026-09-12

## Delivery state

- Pull request: https://github.com/Pin4sf/Waldo-Kennel/pull/110
- Base: `beta` at `0382efcbe7ffce1f949b8801c8fb97edb5475871`
- PR implementation commit: `ea14edeae` (`fix(launch): close outcome execution blockers`)
- Integration merge: `ee6e06659634084fa91b9212d5b85300bdedf926`
- Local branch: `codex/pr110-launch-fixes-20260912`
- Remote PR branch: `codex/kennel-launch-stabilization-20260912`

This handoff records implementation and verification evidence. It does not merge
the pull request, publish a release, deploy Kennel, or create an
`AcceptanceDecision`.

## What changed

- Repository-context settings now have strict PATCH semantics, tri-state limits,
  concurrent atomic persistence, validation, generated API parity, and an
  advanced Settings UI.
- Intake and planning fail closed when repository-context settings cannot be
  read; generated RunBriefs carry the exact Contract criteria, review command,
  check argv, and context-only boundary into execution.
- Replacement Attempt admission, request-key identity, run-state projection,
  proof fingerprint replay, and packaged `KENNEL_RUN_FILE` propagation now keep
  recovery and reconciliation truthful.
- macOS governed checks resolve the system Python runtime without widening
  network authority.
- Packaged Kennel locates its sibling daemon/hook executable, and Codex binary
  symlinks are resolved before spawn so native sidecars such as
  `codex-code-mode-host` remain discoverable.
- The Plan surface separates the text Plan from its graph. Execution renders the
  graph once, then the Attempt lineage/board and current execution controls.
- Attempt cards derive provider branding from their bound `AgentSessionRef`, so
  Codex Attempts no longer render a Claude mark.
- Outcome attention routes into the relevant execution or proof stage and shows
  the daemon-provided reason; ended/unclassified Attempts expose a replacement
  action instead of a dead end.

## Product and authority notes

- Attempt/provider completion remains distinct from Verification and owner
  Acceptance.
- The live test Outcome remains unaccepted.
- The real Attempt 5 returned `needs_you` because its Contract allowed only the
  validator command and did not provide a separate repository-inspection
  capability. That is now surfaced truthfully; no report was fabricated.
- The updater still reports `No published versions on GitHub`. Packaging and
  updater identity are verified, but release publication remains separate work.

See [EXECUTION-LEDGER.md](./EXECUTION-LEDGER.md) for exact commands and live-run
provenance.
