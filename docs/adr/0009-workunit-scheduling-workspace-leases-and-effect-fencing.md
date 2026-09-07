# ADR 0009: WorkUnit Scheduling, Workspace Leases, and Effect Fencing

- **Status:** Accepted
- **Date:** 2026-09-04
- **Scope:** WorkUnit admission, concurrency, workspace isolation/custody, Git integration, consequential effects, restart reconciliation, cleanup
- **Depends on:** ADR 0008 and the existing Outcome/Attempt/Evidence/Acceptance safety model
- **Supersedes:** use of a Project-wide Attempt fence as the long-term execution/concurrency model

## Context

Current `beta` deliberately serializes Outcome Attempts behind a `project:<projectID>` fence. This was a safe foundation while every direct Plan contained one WorkUnit and composed Outcomes focused on governance rather than concurrency.

The canonical v1 requires a WorkUnit DAG with independent branches. Keeping a Project-wide execution fence would make the scheduler artificially serial and would make a Mission Graph that suggests parallel work false.

Simply deleting the fence is also unsafe. Parallel coding work needs explicit workspace ownership, Git/repository integration rules, effect idempotency, restart reconciliation, and cleanup behavior. Provider sessions are not reliable enough to serve as the lock/lease authority.

## Decision

### 1. Schedule WorkUnits, not provider sessions

The scheduler operates on canonical WorkUnits and creates Attempts only after deterministic admission.

A WorkUnit becomes runnable when all required gates pass:

```text
dependencies satisfied
AND required authority exists
AND provider capability exists
AND provider is locally ready
AND workspace lease can be acquired
AND concurrency budget exists
AND no ambiguous/unreconciled prior Attempt exists
AND write/effect constraints permit execution
```

If a gate fails, the WorkUnit remains blocked/waiting/needs-user with an inspectable reason. The daemon never creates an optimistic Attempt and hopes the provider sorts it out.

### 2. Attempt creation follows admission

After admission:

```text
create Attempt
→ allocate/provision WorkspaceLease
→ compile bounded RunBrief
→ launch provider
→ confirm provider-native identity
→ subscribe/observe
→ collect receipt/artifact/evidence candidates
→ reconcile terminal state
→ release/retain workspace according to policy
```

Attempt identity and idempotency are canonical. A retry, recovery, or provider handoff creates a new Attempt rather than silently reusing history.

### 3. WorkspaceLease is durable execution custody

A `WorkspaceLease` conceptually carries:

```text
workspace_id
project_id
outcome_id
work_unit_id
attempt_id
base_revision / base_sha
branch
worktree_path
setup_revision
environment_digest
allocated_ports
write_lease
integration_state
effect_fence/runtime effect context
runtime_processes
preview_endpoints
cleanup_state
created/renewed/released timestamps
```

Exact storage names may evolve, but the semantic boundary is required: workspace ownership cannot exist only in a provider prompt or renderer component.

### 4. Git-backed worktrees are the v1 parallel-write boundary

For repository-backed Projects, independent write-capable WorkUnits use isolated Git worktrees/branches or an equally explicit future workspace implementation.

Workspace provisioning is staged and inspectable:

```text
inspect Project/repository
→ resolve/fetch exact base as needed
→ allocate target path
→ add worktree/branch
→ apply Kennel-owned hooks/setup
→ verify repository/workspace invariants
→ run setup/environment steps
→ record readiness
```

Transient Git lock failures may retry narrowly. Rollback removes only artifacts definitely created by the failed provisioning Attempt. Unknown/dirty debris remains inspectable rather than force-deleted.

### 5. Non-Git folders are usable but not falsely equivalent

The product may open a non-Git folder for Project context, research, planning, terminal work, or a constrained single-writer execution mode.

True parallel writable WorkUnits in v1 require a workspace isolation strategy. The default path is to initialize/use Git before enabling worktree-based parallel writes.

A new Project can be created before Git exists, but the UI/daemon must truthfully expose the limitation and offer Git initialization. Do not invent copy-based parallel workspaces merely to hide the difference.

### 6. Reasoning, writes, integration, and effects have different fences

Do not serialize the whole Project just because two agents exist.

- **read/reason-only WorkUnits:** may run concurrently subject to resource budget.
- **independent repository writes:** may run concurrently in isolated WorkspaceLeases.
- **Git integration/merge/rebase into a shared target:** has a separate controlled integration boundary.
- **consequential external effects:** have separate authority + idempotency/effect fences.

Examples of consequential effects include mutating a PR, deployment, sending a message, charging/spending, destructive external API calls, or other non-worktree state changes.

A worktree lease does not authorize an external effect. An external effect grant does not imply repository integration permission.

### 7. Effect intent/receipt remains separate from provider success

Before a consequential effect, Kennel freezes the intended operation and authority scope. After execution it records/reconciles what actually happened.

If the effect outcome is unknown, Kennel must not repeat it because a provider or daemon restarted. The WorkUnit/Attempt remains blocked/unconfirmed until reconciliation or explicit owner decision.

### 8. Restart/recovery is reconciliation, not respawn-by-default

After renderer, daemon, or provider failure:

- renderer death does not end an Attempt;
- daemon restart reloads canonical Attempts/leases and probes/reconciles provider/workspace facts;
- a surviving provider session is re-associated only when its identity can be established safely;
- a missing/failed probe is not proof the provider died;
- an ambiguous prior Attempt becomes `unknown` / `unconfirmed` rather than `completed` or automatically duplicated;
- only after custody/effect safety is established may policy create a recovery Attempt.

Duplicate Attempts caused solely by restart/retry are a dogfood falsifier.

### 9. Concurrency budget is explicit

Scheduler admission considers a Project/system concurrency budget and provider/runtime limits. The budget is not a provider-brand priority list.

Automatic provider selection, where permitted, chooses only among ready/capable candidates under explicit policy. Explicit user provider choices do not silently fall back.

### 10. Dependencies use retained results, not session liveness

A downstream WorkUnit becomes eligible because upstream dependency requirements are satisfied by canonical terminal/receipt/artifact/proof facts, not because an upstream session process disappeared.

Failed upstream Attempts may be superseded by recovery Attempts without rewriting dependency identity.

A Plan revision can replace topology when strategy materially changes; prior WorkUnits/Attempts remain historical/provenance-bearing.

### 11. Workspace and effect state is explainable

Every blocked/waiting/recovery state must explain the specific gate, such as:

- waiting for WorkUnit A;
- provider installed but not authenticated;
- workspace base changed and requires replan;
- integration lease held by WorkUnit C;
- previous deploy effect outcome unknown;
- cleanup left a dirty worktree requiring inspection.

Do not surface opaque “agent stuck” status when the daemon knows the concrete boundary.

### 12. Cleanup is conservative

Cleanup may automatically remove only Kennel-owned resources whose ownership and safety are established.

Never force-delete:

- dirty registered worktrees with unknown user changes;
- a workspace whose runtime/effect reconciliation is incomplete;
- an unrecognized folder that happens to match a naming pattern.

Cleanup failure has a durable/inspectable state and may become a user decision.

## Consequences

### Benefits

- independent branches can genuinely run concurrently;
- graph UI can represent real scheduler truth;
- provider crashes do not own workspace/authority semantics;
- worktree writes, Git integration, and external effects no longer share one blunt lock;
- restart behavior can be idempotent and explainable;
- single-provider users can still get parallelism by running several independent Attempts of the same provider.

### Costs

- new domain/storage/service state for leases/scheduler/integration/effect reconciliation;
- more explicit provisioning/cleanup failure states;
- additional tests around Git locks, dirty worktrees, daemon restart, provider survival, unknown effects, and concurrent integration;
- non-Git Projects have an honest capability limitation in v1.

## Rejected alternatives

1. **Keep Project-wide serialization.** Rejected because it defeats the execution DAG and makes concurrency UX dishonest.
2. **Remove all fencing and trust worktrees.** Rejected because worktrees do not protect shared Git integration or external effects.
3. **Use provider session/process lifetime as the lease.** Rejected because process probes are ambiguous and providers may survive daemon/renderer failure.
4. **Auto-retry unknown external effects.** Rejected because duplicate consequential effects are worse than explicit uncertainty.
5. **Force-clean dirty worktrees after failures.** Rejected because it can delete unknown user/provider work.
6. **Build a second copy-based workspace system for non-Git folders in v1.** Rejected until dogfood demonstrates enough need to justify a second custody model.

## Implementation boundary

This ADR authorizes the target architecture, not a one-shot rewrite. Implement in verified slices after ADR 0008 domain/persistence work. Preserve current safety/recovery behavior until the new scheduler proves equivalent or stronger invariants.
