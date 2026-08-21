# Upstream provenance and synchronization

Kennel is an independently maintained derivative of [Agent Orchestrator](https://github.com/Untrivial-ai/agent-orchestrator), not a GitHub fork. The initial Kennel repository was created as a squashed/manual copy, so its Git commits do not share ancestry with AO.

## Recorded source

| Field | Value |
| --- | --- |
| Upstream repository | `https://github.com/Untrivial-ai/agent-orchestrator.git` |
| Closest source commit | `66ba38b0fc16c65367a009148fee0cd5afb81a00` |
| Source tree | `d06dd98b8f503b241e9f322ea2353b82442413e3` |
| Source date | 2026-08-17 |
| License | Apache-2.0 |
| Kennel snapshot audited | `b1bcd37fd0911a86da5bb21c07ac3a0b9adecb3b` |

This base was independently checked on 2026-08-18 against the latest 60 commits then reachable from AO `main`. It was the unique closest tree: 2,406 files had the exact same blob at the same path, across 2,468 shared paths. The two repositories had no merge base. Machine-readable values live in [`upstream.json`](../upstream.json), and [`scripts/compare-upstream.sh`](../scripts/compare-upstream.sh) reproduces the tree comparison without changing the worktree.

## Modification boundary

AO remains the chassis for provider processes, worktrees, terminals, streaming, session lifecycle, recovery, and Claude/Codex adapters. Kennel-owned changes begin at product identity and local-state isolation, foundation/security policy, and the prototype Outcome overlay already present in this repository. Planned Waldo Mission, memory, personal-agent, authority, verification, and related semantics are not part of this foundation and must not be inferred from the AO chassis.

The Go module path remains `github.com/aoagents/agent-orchestrator/backend` for now. It is a source-level compatibility seam used by thousands of internal imports; it is not a packaged application identity, network destination, update target, or claim that this repository is AO. A separate module-path migration should happen only with a measured downstream-consumer inventory because it produces a repository-wide patch with no runtime-isolation benefit.

## Non-destructive synchronization procedure

1. Start from a clean Kennel branch. Never sync directly on `main`.
2. Add or refresh the AO remote:

   ```sh
   git remote add ao-upstream https://github.com/Untrivial-ai/agent-orchestrator.git 2>/dev/null || \
     git remote set-url ao-upstream https://github.com/Untrivial-ai/agent-orchestrator.git
   git fetch --no-tags ao-upstream main
   ```

3. Reproduce the recorded baseline with `./scripts/compare-upstream.sh`.
4. Choose the reviewed AO range explicitly. Inspect it before applying anything:

   ```sh
   git log --oneline 66ba38b0fc16c65367a009148fee0cd5afb81a00..ao-upstream/main
   git diff --stat 66ba38b0fc16c65367a009148fee0cd5afb81a00..ao-upstream/main
   ```

5. Port selected commits or patches into a normal Kennel feature branch. Preserve Kennel identity, state, updater, packaging, migration, and product-language contracts. Do not merge AO wholesale.
6. If an upstream migration number collides, preserve both shipped ledgers and add an idempotent physical-schema reconciliation test. Kennel's historical `0098_project_chat_assistant_projection.sql` and AO's later, different `0098_session_native_identity_generation.sql` are the first recorded example.
7. Update `upstream.json` only after tests and a review identify the new audited base. Run the full foundation matrix and the packaged-identity assertion.

This workflow produces ordinary reviewable commits. It does not graft histories, rewrite `main`, or force-push.

## Optional ancestry repair plan — approval required

A faithful ancestry repair would require creating a replacement history rooted at AO, replaying the two published Kennel commits and all later Kennel work, verifying tree equivalence and tags, coordinating a freeze, and replacing the published default branch. That replacement would change every Kennel commit ID and require a force update of `main`. It is therefore explicitly outside this foundation branch and must not be attempted without separate user approval, maintainer coordination, backups, and a tested rollback ref.

The non-destructive sync seam above is the active maintenance method until that decision is made.
