# ADR 0003: Local-first Waldo Core inside Kennel

- Status: Accepted
- Date: 2026-08-20
- Scope: Kennel v1 launch topology and post-attachment custody

## Context

Kennel's launch wedge is agent-heavy Mac users who need to move from a stated Outcome to verified acceptance with less coordination and supervision. Requiring a hosted Waldo backend, account, API service, and Waldo-funded model calls would add account friction, infrastructure cost, privacy exposure, and a second failure domain before that product loop is proved.

Waldo and Kennel still require a semantic boundary. Waldo owns responsibility and control-plane meaning; Kennel owns local execution and custody. That boundary does not require separate deployments in v1.

## Decision

Kennel v1 ships a **Local Waldo Core** inside the existing Go daemon. Electron remains a thin supervisor and user interface over the loopback HTTP API.

The Local Waldo Core owns:

- Outcome and ContractRevision semantics;
- optional versioned PlanRevision, presented as the Mission Map;
- Work Unit compilation and authority requirements;
- decision, evidence, verification, acceptance, and successor lineage;
- orchestration policy and provider-specific RunBrief compilation.

The Kennel Runtime owns:

- local projects, workspace bytes, worktrees, processes, terminals, and browser sessions;
- provider authentication and provider AgentSessions;
- raw traces, unselected artifacts, local observations, and recovery facts;
- enforcement of the locally admitted plan and capability grants.

The daemon's local SQLite database is the sole canonical writer for v1. The renderer does not own durable product logic or write around daemon service boundaries.

Launch does not require:

- a hosted Waldo backend or synchronization service;
- a Waldo account;
- a Waldo-operated LLM API or Waldo-funded inference.

Where technically and contractually supported, planning, execution, critique, and summarization use the user's authenticated provider sessions. Model output proposes plans and interpretations; deterministic local code owns validation, authorization, state transitions, and acceptance gates.

## Optional hosted attachment

A future Waldo account may be recommended when it provides concrete backup, continuity, cross-device, or relationship value, but it remains optional. Signing in alone never uploads local state or transfers authority.

Attachment is an explicit Project/workspace action:

- future Outcomes in the attached Project inherit attachment;
- historical Outcomes remain local unless the attachment flow presents them and the user selects them;
- hosted Waldo may become canonical for Project identity, Outcome contracts, PlanRevisions/Work Units, authority grants, decisions, evidence metadata and digests, verification results, acceptance, and follow-up lineage;
- Kennel remains canonical for workspace bytes, worktrees, terminals, raw traces, credentials, provider authentication, and unselected artifacts;
- raw artifacts, excerpts, or traces cross the boundary only through explicit disclosure or learning permission.

The attachment protocol must establish one canonical authority. Dual canonical writers are forbidden. Offline commands, acknowledgement, conflict behavior, detach, revoke, and deletion remain separate design decisions.

## Consequences

### Benefits

- proves the launch loop without cloud infrastructure or forced sign-up;
- preserves local custody and offline usefulness;
- reuses the AO-derived daemon, SQLite, worktree, session, terminal, browser, and provider-adapter chassis;
- keeps a stable Waldo/Kennel responsibility boundary that can later span local and hosted deployments.

### Costs and deferred capabilities

- no cross-device continuity, hosted recovery, or remote proactive execution at launch;
- no centrally operated fallback model unless added later;
- provider subscription terms, unattended execution support, and capability parity must be verified provider by provider;
- a later attachment requires an explicit migration/synchronization protocol rather than implicit dual-write sync.

## Superseded assumptions

This decision supersedes any earlier assumption that Kennel v1 requires an account, hosted Waldo authority, or a Waldo-funded LLM call. It does not supersede the post-attachment custody boundary; it makes that boundary a later explicit deployment mode.
