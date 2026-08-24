# Hermes and open-source Grok Bot alternatives for Waldo Home

- **Date:** 2026-08-23
- **Status:** Architecture and product research only; no implementation authority
- **Scope:** Current Hermes Agent updates and source-verifiable open-source systems relevant to Waldo Home, conversation, agents, routines, approvals, execution, and continuity
- **Issue boundary:** This research does not expand issue #23. Personal Home persistence, durable Quick Capture, and explicitly confirmed Open Loops remain the issue #23 corridor. Agent chat, routines, memory, computer use, and multi-agent coordination require separately planned work.

## Executive finding

The reference systems converge on a sound runtime shape but not on Waldo's product model:

1. A durable backend or gateway owns sessions, agents, schedules, approvals, and execution; the renderer only presents state and sends intent.
2. Persistent specialists are isolated by profile, workspace, memory, credentials, and session history.
3. Short-lived subagents are different from durable assistants and need bounded depth, concurrency, budget, and lifetime.
4. Consequential actions need an explicit approval object bound to the exact execution plan, not a permissive chat interpretation.
5. Schedules run outside the model and keep durable definitions, state, and run history.
6. Agent memory is useful but fallible. It must remain inspectable and correctable and must not become responsibility truth.

Waldo should adopt these mechanisms while rejecting the dominant product metaphor of a roster of equally authoritative bots. The recommended shape is **one Waldo relationship, with bounded specialists and task-scoped workers underneath it**. Home stays an adaptive brief with one expanded Quick Capture; conversation is a globally reachable interaction surface, and agent execution is a fact-backed projection, not the Home information architecture.

## Evidence discipline

The facts below come from official repositories, repository documentation, and release records as of 2026-08-23. README and release claims establish described behavior, not independent reliability or production quality. Rakazo, Guaca, and OpenMausBot are moving quickly and should be treated as mechanism references rather than proven product foundations. Recommendations for Waldo are explicitly labeled as interpretation.

## Latest Hermes update

### Observed

- The latest tagged release is **Hermes Agent v0.20.5 (`v2026.8.19`)**, published 2026-08-21. It is a patch rollup over v0.20.4 and reports Bot Mode room work, conversation summaries, file/PDF attachments, runtime stall guards, and cron jobs gaining persistent memory and per-job reasoning effort. [v0.20.5 release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.19)
- At inspection, `main` was 215 commits ahead of v0.20.5. Some current documentation and source behavior is therefore unreleased and must not be represented as part of the stable tag. [v0.20.5 to main comparison](https://github.com/NousResearch/hermes-agent/compare/v2026.8.19...main)
- The major current feature baseline arrived in **v0.20.0**, including streaming voice and barge-in, signed outbound webhooks, sandboxed/versioned desktop artifacts, A2A v1.0, mid-turn redirect, source-grounded research, smarter context compression, and expanded approval controls. [v0.20.0 release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.3)
- Bot Mode is a UI over Hermes profiles. Each Bot has isolated configuration, memory, skills, credentials, and chat history under its own profile directory. A canonical Bot Chat is persistent; routines are ordinary cron jobs namespaced to that profile. [Bot Mode](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/bot-mode.md)
- The latest Bot Mode supports a `SESSIONS | BOTS` split, a durable roster, transient `Active now` presence, hidden bots, canonical bot chats, two-to-six-member group rooms, capped rounds, `Needs You`, bot-to-bot messaging, and cross-machine routing. [Bot Mode](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/bot-mode.md), [v0.20.4 release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.18)
- Current Bot rooms are an architectural warning for Kennel: orchestration and the complete room log live in a desktop plugin, with only a bounded projection mirrored through profile metadata. That conflicts with Kennel's thin-renderer and daemon-truth boundaries even though the resulting UI is a useful benchmark. [Bot renderer orchestration](https://github.com/NousResearch/hermes-agent/blob/main/apps/desktop/src/plugins/hermes-bots/plugin.js)
- Hermes' renderer is not the durable owner. Its system has CLI, gateway, API, desktop, and editor entry points over an agent loop, SQLite session persistence with FTS5 and lineage, a messaging gateway, plugins, terminal backends, and a cron subsystem. [Architecture](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/developer-guide/architecture.md)
- Session truth lives in SQLite/WAL with sessions, messages, tool calls, FTS5, compression lineage, usage attribution, routing, locks, and delegation records. [Session storage](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/developer-guide/session-storage.md)
- Subagent delegation uses isolated contexts and terminals and defaults to bounded parallelism. It is explicitly **not durable**: interruption of the parent can cancel the children. Hermes directs long-lived work to cron or a background terminal process instead. [Delegation](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/delegation.md)
- Hermes cron jobs start fresh agent sessions, persist job definitions and output, support pause/resume/edit/run/remove, and scan prompts for known injection and credential-exfiltration patterns. Definitions remain in atomic JSON, while attempts have a SQLite ledger with immutable terminal states; a crash with uncertain side effects becomes `unknown` rather than being automatically replayed. [Cron](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/cron.md)
- Hermes separates identity and context files: `SOUL.md` describes the agent, `USER.md` the user, `MEMORY.md` learned material, and project context files project rules. Memory is injected as a frozen session-start snapshot and can be write-gated or edited. [Which file does what](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/which-file-does-what.md)
- Built-in memory is deliberately bounded Markdown, but agent and background-review writes are automatic unless `memory.write_approval` is enabled. Waldo should invert that default for personal truth: staged, source-bearing review first. [Memory](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/memory.md)
- Hermes also distinguishes non-durable delegated children from a durable SQLite Kanban queue with claims, idempotency, review, handoffs, and restart recovery. This is a useful proof that chat delegation and durable work are different contracts. [Kanban](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/kanban.md)
- Hermes' own security policy says only OS isolation is a true adversarial boundary; in-process scanners, allowlists, redaction, plugins, and skills are heuristic controls. Its computer-use path keeps existing signed-in browser profiles inaccessible unless a bounded capability manifest is reviewed. [Security policy](https://github.com/NousResearch/hermes-agent/blob/main/SECURITY.md), [computer-use permissions](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/computer-use.md#permission-modes-and-logged-in-browser-profiles)

### Interpretation for Waldo

Hermes demonstrates that a rich Bot UI can be built as a projection over existing durable profile, session, and cron primitives. Waldo should use the same architectural direction—daemon-owned primitives and thin UI—but not the same identity model. A Hermes Bot is an autonomous profile. A Waldo specialist should be an executor operating within Waldo's user-owned authority and an Outcome or Work Unit scope.

Hermes' `Active now` strip and `Needs You` state are better Home references than its full bot roster. They reveal work only when it changes the user's next decision. The roster, model picker, credentials, skills, and machine placement belong in Work or an inspectable advanced surface.

## Open-source Grok Bot alternatives

### 1. Rakazo — closest broad product analogue

**Observed:** Rakazo describes persistent bots with their own conversation, memory, routines, history, browser, terminal, files, and graphical computer. It distinguishes shared Team Computers from isolated Private computers and supports both durable peer bots and short-lived subagents. Its web, Electron, and Expo clients use one API. The monorepo separates domain, contracts, persistence, adapters, UI, API, worker, and infrastructure packages. Postgres/Prisma and Graphile Worker provide persistence and background execution. [Rakazo README](https://github.com/elie222/rakazo), [computer runtime and isolation](https://github.com/elie222/rakazo/blob/main/docs/computer-runtime.md)

Rakazo supports Docker, remote sandboxes, and a trusted local-computer provider. Its own documentation calls direct host execution the least isolated choice and suitable only for trusted single-user local operation. Its testing matrix distinguishes deterministic fake runtimes, integration tests, topology recovery, and explicit live-sandbox acceptance. [Rakazo README](https://github.com/elie222/rakazo)

**Adopt:** explicit durable-bot versus short-lived-subagent types; API-shared clients; queue-backed recovery; real-provider acceptance tests behind deterministic contract tests; explicit runtime isolation labels.

**Reject:** making the roster the primary Home; treating each bot's memory or conversation as independent canonical truth; adopting its TypeScript/Postgres stack when Kennel's Go/SQLite daemon already supplies the correct local architecture.

### 2. OpenMausBot — closest thin local harness

**Observed:** OpenMausBot presents a local chat roster over installed Claude, Codex, and Grok CLIs. A loopback harness server owns provider processes, a registry, a permission broker, a fan-in event bus, HTTP commands, and one SSE stream. Provider-native protocols are normalized into canonical runtime events and logged per thread as NDJSON; the app has one server-backed reducer and no client-side provider transports. [OpenMausBot README](https://github.com/milind-soni/OpenMausBot)

Its bots can use cloud computers, isolated local VMs, or the host machine. Shell, edit, and question approvals appear inline. Provider drivers degrade to unavailable instead of crashing the fleet. As of the current README, the core message-to-tools-to-approval loop is described as working, while routines remain a placeholder. [OpenMausBot README](https://github.com/milind-soni/OpenMausBot)

**Adopt:** provider normalization behind a daemon contract; one event stream into a thin renderer; unavailable-provider states; inline approvals and computer inspection.

**Adapt:** NDJSON runtime events can be useful execution telemetry, but Waldo's canonical state belongs in typed SQLite tables with trigger-backed CDC. Provider events must never directly create or dispose responsibility.

### 3. Guaca — clearest multi-agent guardrails

**Observed:** Guaca keeps its asynchronous agent runtime in Rust rather than the webview. Each agent is a task with an inbox; the frontend renders state and forwards intent. It files every message to one canonical channel at write time and derives the activity view from the same table. [Guaca architecture](https://github.com/madebywelch/guaca/blob/main/docs/ARCHITECTURE.md)

Guaca has separate limits for model calls per run, relay depth, pairwise messages, recipients per send, and identical repeated messages. A run snapshots its limits so policy changes cannot mutate the budget mid-run. Limit failures are visible rather than silently dropped. External actions and agent creation stop for approval; its README intentionally omits a standing `Always allow` path for external actions. [Guaca architecture](https://github.com/madebywelch/guaca/blob/main/docs/ARCHITECTURE.md), [Guaca README](https://github.com/madebywelch/guaca)

Its persistent memory is one editable agent-maintained Markdown file, and its project documentation is candid that local credentials are currently plaintext despite restrictive filesystem permissions. This is useful evidence for both inspectability and the danger of mistaking readable storage for sufficient secret handling. [Guaca README](https://github.com/madebywelch/guaca)

**Adopt:** separate runaway guards; snapshot policy at run start; visible refusal reasons; derive activity from canonical events; never persist hidden reasoning.

**Reject:** free-form Markdown as Waldo's canonical memory or responsibility state; group-wide credential injection; a team-chat transcript as the organizing model for Home.

### 4. OpenClaw — strongest durable gateway/control reference

**Observed:** OpenClaw runs isolated agents in one Gateway, each with its own workspace, state directory, auth profile, and SQLite-backed session history. Inbound channel accounts route deterministically through explicit bindings. Cross-agent memory search is disabled by default; intentionally shared material requires an explicit shared path. [Multi-agent routing](https://github.com/openclaw/openclaw/blob/main/docs/concepts/multi-agent.md)

Its approval flow records a canonical system run plan before approval. The approved execution reuses that stored command, working directory, agent, and session binding; a later mismatch is rejected. Per-agent allowlists prevent approvals from leaking across agents. [Exec approvals](https://github.com/openclaw/openclaw/blob/main/docs/tools/exec-approvals.md)

Automations run in the Gateway, not the model. Definitions, runtime state, background-task records, and run history persist in SQLite across restart. Unknown completion and required-delivery failure remain inspectable rather than being silently treated as success. Current authorization is narrowed at run time, and revoked capabilities fail closed. [Automations](https://github.com/openclaw/openclaw/blob/main/docs/automation/cron-jobs.md)

**Adopt:** frozen approval plans; deterministic routing; per-agent capability boundaries; durable job and run ledgers; restart recovery; fail-closed revalidation.

**Adapt:** OpenClaw agents are full identities. Waldo's providers and specialists should remain subordinate to one Waldo identity and one responsibility model.

### 5. Supporting references, not direct Home templates

- **Agent Zero** separates projects, profiles, skills, model presets, memory, and subordinate agents. Its own memory guide says long-term memory remains unsolved and can preserve stale assumptions or false confidence. This strongly supports inspectable, user-correctable memory rather than automatic authority. [Agent Zero](https://github.com/agent0ai/agent-zero), [memory guide](https://github.com/agent0ai/agent-zero/blob/main/docs/guides/memory.md)
- **Suna/Kortix** is useful for per-session sandbox and branch isolation and for treating the sandbox as execution truth while syncing a read model. It is a hosted management system, not a Personal Home model. [Suna](https://github.com/kortix-ai/suna), [manifesto](https://github.com/kortix-ai/suna/blob/main/MANIFESTO.md)
- **Paperclip** is a control plane: durable tasks, atomic checkout, execution adapters, run identity, and evidence/cost/session capture. It should inform Work orchestration, not turn Home into a company dashboard or make a task row responsibility truth. [Architecture](https://github.com/paperclipai/paperclip/blob/master/docs/start/architecture.md), [execution semantics](https://github.com/paperclipai/paperclip/blob/master/doc/execution-semantics.md)

## Mechanism comparison

| Concern | Strongest reference mechanism | Waldo decision |
|---|---|---|
| Durable identity | Hermes profile; OpenClaw isolated agent | One durable Waldo identity; optional specialists are scoped executors |
| Conversation | Canonical Bot Chat; per-thread harness events | Durable Waldo conversation plus typed native commands |
| Specialist work | Durable peer bot and short-lived subagent are separate | Persist `AgentProfile` only when user-pinned; default to task-scoped `AgentRun` |
| Runtime ownership | Gateway/harness owns processes and events | Kennel daemon owns provider sessions and lifecycle |
| Renderer | Server-backed reducer, intent forwarding | Keep Electron/React as a thin projection and command surface |
| Responsibility | References generally use tasks/messages | Only canonical Responsibility/Open Loop commands can create or dispose responsibility |
| Approvals | Frozen run plan plus scoped allowlist | Persist exact proposed effect, parameters, scope, hash, expiry, and disposition |
| Routines | Gateway-owned durable scheduler and run history | Versioned daemon-owned routine definition plus append-only run ledger |
| Memory | Profile Markdown/DB; explicit warnings about fallibility | Source-bearing proposals with correction and deletion; never implicit responsibility |
| Attention | Active now, waiting-user, unread, run state | Derived `Needs You`, `Working`, `Waiting`, `Ready to review`, `Failed` from durable facts |
| Isolation | Per-profile state; per-bot computer; per-session sandbox | Capability-scoped worktree/sandbox; no ambient shared credentials |
| Recovery | Worker queue, SQLite sessions/runs, persisted jobs | Idempotency keys, revisions, restart recovery, trigger-backed CDC |
| Safety limits | Guaca's independent run, depth, pair, fanout limits | Snapshot bounded execution policy per run and surface every refusal |
| Evidence | Inline tools, artifacts, screenshots, logs | Artifacts are evidence candidates; user acceptance remains separate |

## Recommended Waldo architecture

### 1. Keep four planes separate

```text
Personal truth plane
  PersonalHome -> Capture -> Candidate -> explicit confirmation -> Open Loop

Conversation plane
  Waldo Thread -> Messages -> Questions -> Proposed Commands

Execution plane
  Work Unit -> Agent Run -> Tool/Computer Events -> Artifacts/Evidence

Control plane
  Approval -> frozen Effect Plan -> Execution -> Result -> append-only disposition
```

The renderer can join these projections for comprehension, but the database and services must not collapse them. In particular:

- a message is not an Open Loop;
- an agent plan is not authorization;
- an approval is not proof of completion;
- provider completion is not Outcome acceptance;
- a screenshot or artifact is evidence, not truth by itself;
- memory recall is context, not responsibility ownership.

### 2. Use existing Kennel technology

Do not change stacks to imitate the references. The best fit remains:

- **Go daemon** for controllers, services, ports, process supervision, scheduling, authority checks, and recovery;
- **SQLite** for local canonical state, revisions, idempotency, append-only dispositions, job/run ledgers, and trigger-backed `change_log` CDC;
- **generated OpenAPI and TypeScript contracts** for a strict renderer boundary;
- **Electron + React** for Home, chat, approvals, and inspector projections;
- **provider adapters** for Codex, Claude, Grok, or future engines, normalized behind Kennel's own event vocabulary;
- **isolated worktrees/sandboxes** for consequential Work execution, with host access separately and explicitly granted.

The architecture to copy is the boundary discipline, not Rakazo's TypeScript/Postgres, Hermes' Python/JSON cron store, or Guaca's Rust runtime.

### 3. Proposed later domain contracts

These are planning names, not authorization to implement them in issue #23:

- `WaldoConversation` and `ConversationEvent`
- `AgentProfile` for a user-pinned durable specialist
- `AgentRun` for a task-scoped or profile-backed execution attempt
- `EffectPlan` containing canonical operation, parameters, target, capability scope, preconditions, and digest
- `ApprovalRequest` bound to one `EffectPlan` revision, with expiry and append-only decision events
- `RoutineDefinition`, immutable `RoutineRevision`, and `RoutineRun`
- `ArtifactRef`, `EvidenceRef`, and `VerificationRecord`
- derived `AttentionItem`, never stored as an independent source of truth

Every write command should carry an idempotency key and expected revision. Approval execution must verify that the stored plan digest and current preconditions still match before any effect occurs.

## Recommended Home interaction

### Preserve

- The merged adaptive Today brief and its calm Home aesthetic.
- The global horizontal Home/Work pill and lane-specific navigation.
- Exactly one expanded Quick Capture on Today.
- Wide and narrow layouts, keyboard navigation, and focus restoration.
- Provenance, explicit uncertainty, and unavailable-facts states.

### Add later, without replacing Home

- A globally reachable **Talk to Waldo** surface, opened from the shell or keyboard, that becomes a full conversation on narrow windows and a side sheet or focused layer on wide windows.
- A conditional **Waldo is helping** projection only when work is active. It should show the Outcome/Open Loop scope, current factual state, last evidence, and stop/inspect action—not a stream of decorative agent activity.
- A **Needs You** section only for concrete questions, approvals, conflicts, or review decisions. Each card states what is blocked, the exact proposed action, consequence, evidence, safe default, and expiry.
- Specialist identity in Work: the selected Outcome can expose its current agent, work unit, provider session, artifacts, and handoffs. Home may mention the specialist only as provenance.
- A routine result as a brief item or Waldo message only after a durable run exists. Failure, unknown completion, or missing facts must remain explicit.

### Do not add

- A permanent Grok-style roster as the primary Home sidebar.
- Agent avatars, presence, or typing activity without a user decision or scoped work item.
- Automatic Open Loop creation from chat, memory, routines, observation, tool output, provider completion, or suggestion clicks.
- Shared ambient credentials across specialists.
- Standing approval for broad external effects.
- Plausible fixture data on canonical desktop routes when the daemon is unavailable.

## Sequenced product plan

### Now: issue #23 only

1. Daemon-owned `PersonalHome` read model.
2. Durable Quick Capture with idempotency and restart read-back.
3. Candidate state that remains non-canonical until explicit confirmation.
4. Explicit Open Loop confirmation using canonical responsibility identity.
5. Append-only disposition and revision-conflict handling.
6. Thin renderer integration using the merged Home components and honest unavailable states.

Do not introduce `AgentProfile`, routines, memory, agent chat, computer use, or new provider behavior in this issue.

### Next separately planned slice: Waldo conversation

1. Durable conversation/events owned by the daemon.
2. Native proposed-command envelope distinct from assistant text.
3. Capture and correction commands that reuse Personal Home services.
4. Explicit confirmation UI that invokes the canonical command.
5. Search, provenance, narrow/wide behavior, focus restoration, and restart recovery.

### Later Work slice: bounded agent execution

1. Provider-neutral `AgentRun` and event vocabulary.
2. Work Unit/Outcome scoping and isolated execution.
3. Frozen Effect Plan and approval corridor.
4. Stop, redirect, retry, recovery, evidence, and verification.
5. Optional durable specialist profiles after task-scoped execution is trustworthy.

### Later automation slice

1. Versioned routine definition and explicit authority ceiling.
2. Durable scheduler, idempotent run creation, lease/recovery, and run history.
3. Fresh-context execution with explicit attached sources/skills.
4. Revalidation of credentials and capabilities at run time.
5. Fail-closed delivery, pause/revoke, evidence, and user-visible unknown states.

## Decision

Use **Waldo-led, agent-assisted Home**:

- Waldo is the one durable relationship.
- Home is an adaptive brief plus direct capture and decisions.
- Chat is a command-and-explanation channel, not the source of responsibility truth.
- Specialists are mostly task-scoped and visible under Work.
- A durable specialist exists only when the user intentionally creates or pins one.
- Home shows agent activity only when it changes what the user needs to know or decide.

This preserves what is distinctive about Waldo—user-owned continuity, explicit responsibility, inspectable truth, and conscious closure—while adopting the best runtime and interaction mechanisms demonstrated by Hermes, OpenClaw, Rakazo, OpenMausBot, Guaca, Agent Zero, Suna, and Paperclip.
