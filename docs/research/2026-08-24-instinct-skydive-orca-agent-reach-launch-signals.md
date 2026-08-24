# Instinct, Skydive, Orca, and Agent-Reach — Waldo launch signals

**Date:** 2026-08-24

**Status:** Product and architecture research; no implementation, issue, PR, release, or deployment authorization

**Scope:** Classify four current external signals, extract clean-room product and engineering lessons, and translate them into a narrow Waldo/Kennel launch sequence

**Current Waldo baseline:** `origin/beta@b57413b86f7185c2537905375d23ea7bb5d6ab5e`

## Executive conclusion

These references strengthen the urgency to launch, but they do not justify a breadth race.

- **Instinct** is the closest category signal: a low-interface consumer personal assistant that claims proactive follow-through and broad computer/account access. It remains private-access, and no first-party evidence establishes a large upcoming launch, traction, reliability, or product-market fit.
- **Skydive** is an already packaged cloud-agent workforce for startup teams. It validates outcome-first setup, cross-channel identity, scheduled work, credential isolation, and explicit local-computer grants, but it is not the same user-owned personal-life category as Waldo.
- **Orca** is a strong reference for Kennel's work harness: isolated worktrees, recovery, mobile steering, design-context capture, diff review, and remote execution. It is not a Waldo identity or personal-agency model.
- **Agent-Reach** is a useful capability-router reference: channel registry, preferred/fallback backends, health checks, and repair guidance. Its direct use of upstream CLIs, cookies, and mutable installers is too broad for Waldo's authority and audit boundary.

The launch target should be one real, inspectable loop:

`capture -> confirmed responsibility -> context -> bounded plan -> exact permission -> action -> evidence -> conscious close`

Waldo's defendable position is not governance without agent capability, and it is not an integration-count race. Waldo is both an active personal agent and the user-facing coordinator of an orchestration platform. It should eventually match the useful agent primitives the category is converging on—custom agents, teams, tools, computers, memory, routines, handoffs, and multi-surface continuity—while applying Waldo's stronger responsibility and proof model to all of them.

The position is:

> Waldo is one user-owned personal agent across life and work, and the orchestration relationship through which the user creates, equips, coordinates, and supervises other agents. Kennel is its desktop execution presence. Every agent can receive the memory, tools, team membership, and isolated computer its job requires, while Waldo preserves continuity, authority, evidence, correction, acceptance, and return to the user.

## Evidence discipline

This report uses the following labels:

- **Observed** — directly present in a first-party page, documentation, repository, code, or legal policy at the cited snapshot.
- **Author-claimed** — stated by the product owner but not independently reproduced here.
- **Secondary-reported** — reported by another company, publication, or social source and not independently established.
- **Inference** — a bounded conclusion drawn from the observed evidence.
- **Unknown** — material behavior not established by the inspected sources.
- **Proposed** — a Waldo/Kennel decision for review; not shipped functionality.

Product marketing is evidence of positioning, not reliability, adoption, retention, or safety. Repository code is evidence of an implementation mechanism, not proof that every packaged or hosted path behaves correctly in production.

The supplied [Carly article](https://www.usecarly.com/blog/what-is-instinct-ai/) is Carly-authored competitor marketing. It is a useful launch-week social-signal summary and is first-party evidence for Carly's framing, but only secondary evidence about Instinct. Its anecdotes are not treated as Instinct traction or benchmark results.

## Source and reuse boundary

| Reference | Primary source boundary | Reuse boundary | Qualification |
| --- | --- | --- | --- |
| Instinct | [Homepage](https://instinct.co/), [Privacy Notice](https://instinct.co/privacy-policy), and [Terms](https://instinct.co/terms) | Proprietary. Terms prohibit reverse engineering, benchmarking, and use to create competing products. Use only public positioning and legal-contract observations. | Closest category signal; private-access. |
| Skydive | [Product site](https://www.skydive.com/start) and [official documentation](https://www.skydive.com/docs/agents/memory) | Proprietary product. Extract public behavior and architecture lessons without reconstructing unpublished implementation. | Publicly packaged B2B cloud-agent platform. |
| Orca | [`stablyai/orca@55258f34`](https://github.com/stablyai/orca/tree/55258f34ad4b94cb3bdda926f65bc5f228680a6e) | [MIT](https://github.com/stablyai/orca/blob/55258f34ad4b94cb3bdda926f65bc5f228680a6e/LICENSE); preserve notice for copied/adapted code and audit dependency licenses separately. | Open-source coding-agent ADE; fast-moving main branch. |
| Agent-Reach | [`Panniantong/Agent-Reach@93ae1d18`](https://github.com/Panniantong/Agent-Reach/tree/93ae1d18c37b707dec053c7c4f9d91cd8ef8943d) | [MIT](https://github.com/Panniantong/Agent-Reach/blob/93ae1d18c37b707dec053c7c4f9d91cd8ef8943d/LICENSE); upstream tools retain independent licenses and platform terms. | Python package `1.5.0`, classified Beta in its [package metadata](https://github.com/Panniantong/Agent-Reach/blob/93ae1d18c37b707dec053c7c4f9d91cd8ef8943d/pyproject.toml). |

## 1. Instinct: closest category signal

### What is established

**Observed / author-claimed:** Instinct presents one personal assistant connected to applications and devices, including email, messaging, screen, audio, and location. It says users can text or call it, and that it can proactively follow up and act through a phone and computer. Public access remains limited to a waitlist or member invitation while Instinct says it is scaling compute. No public price or general-availability date is established on the [official homepage](https://instinct.co/).

**Observed legal contract:** Instinct's [Privacy Notice](https://instinct.co/privacy-policy) describes macOS and mobile applications and an autonomous assistant that, when engaged, may access screen/application content, documents, private messages and email, optional audio, credentials, payment data, health-related information, clicks, keystrokes, cursor position, and location. Google Workspace API data has an explicit no-training/no-advertising exception; an equivalent blanket exception is not stated for all other input classes.

**Observed authority boundary:** The [Terms](https://instinct.co/terms) authorize actions, commitments, agreements, and transactions the service deems responsive to user input. They say confirmation safeguards may not prevent unintended actions and place monitoring and consequences on the user. The privacy notice itself recognizes unintended payments or communications, sensitive-data disclosure, and hidden-instruction manipulation as risks.

### What is not established

- Execution topology, model/provider stack, and credential isolation.
- Which consequential actions always require explicit approval.
- Evidence, audit, rollback, reconciliation, and recovery behavior.
- Non-Google retention and derived-memory deletion semantics.
- Prompt-injection defenses and measured task reliability.
- Active users, retention, engagement, pricing, or a general launch date.

The Carly article's life-administration success and failure stories remain **secondary-reported** anecdotes, not controlled evidence.

### Waldo decision

**Adopt**

- One continuous personal relationship reached through low-friction surfaces.
- Follow-through on dropped threads and unfinished responsibilities.
- Proactive contact when a responsibility materially changes.
- Demos centered on completed outcomes rather than chat quality.

**Adapt**

- Broad sensors become optional, purpose-bound `CaptureGrant`s with visible state, exclusions, pause, revoke, retention, export, and deletion.
- A request becomes an inspectable episode, Open Loop, proposal, or bounded run—not blanket delegated authority.
- Every consequential action crosses a typed permission boundary and returns evidence plus an unknown-outcome path.

**Reject**

- Always-on observation as a default.
- “Act as deemed responsive” authority for purchases, messages, commitments, deletion, or account changes.
- Broad irreversible training rights over personal material.
- User monitoring and liability transfer presented as an adequate safety model.

## 2. Skydive: launched cloud workforce, adjacent competitor

### What is established

**Observed:** Skydive is publicly packaged as a team-oriented cloud-agent platform, with web, desktop, CLI, API, documentation, and paid plans on its [start page](https://www.skydive.com/start). Public packaging does not establish usage, reliability, or retention.

**Author-claimed mechanisms:**

- One agent identity and memory across web, Slack, email, iMessage, desktop, and terminal, with a cloud computer, browser, filesystem, and routines.
- [Durable file-based memory](https://www.skydive.com/docs/agents/memory) shared at agent/team level, written by the agent, retrieved per conversation, editable/forgettable, and stored in the agent's Git repository.
- [Scheduled runs](https://www.skydive.com/docs/agents/scheduled-runs) for recurring, one-time, and periodic work. Suggested routines require setup; active schedules can be paused, resumed, changed, or cancelled, and keep run history linked to conversations.
- A [transparent credential proxy](https://www.skydive.com/docs/integrations/permissioning) that applies credentials outside prompts and agent code, with service/resource/permission scoping and revocation.
- A default-off, per-agent [Portal](https://www.skydive.com/docs/capabilities/portal) granting revocable access from a cloud agent to a local browser, files, tools, or localhost.

### Conflicts and unknowns

- Marketing says no API or OAuth integration needs configuring, while documentation describes connectors, proxy-injected tokens, MCP, and APIs. The mechanism appears hybrid and should be named precisely.
- Marketing claims such as “anything a human can do,” instant setup, and permanent correction are not reliability benchmarks.
- Memory admission, multi-user conflict resolution, correction propagation, and deletion propagation are not established.
- Consequential-action approval, effect reconciliation, and rollback are not sufficiently established for Waldo's trust boundary.
- Independent sandbox/security validation, browser success rates, customer count, retention, and product-market fit remain unknown.

### Waldo decision

**Adopt**

- Outcome-first setup rather than forcing users to design workflows.
- One identity across surfaces.
- User-created persistent agents and explicit agent teams.
- Agent-to-agent delegation and handoff under one visible coordinator.
- A separate sandbox/computer for an agent when its task requires stateful execution.
- Visible activity, intervention, routine history, pause, and revoke.
- Credentials outside the agent's prompt/context.
- Default-deny access to the user's local machine.

**Adapt**

- Keep Waldo as the user-side personal relationship and top-level coordinator while allowing specialists, temporary subagents, and explicit teams to be capable agents in their own right.
- Allow a team to have an internal lead or coordination policy for a bounded run, while Waldo continues to own the user handoff, effective authority ceiling, and final return path.
- Model an agent's computer as an explicit execution-environment attachment with lifecycle, scope, cost, persistence, credential, recovery, and deletion policy—not as the agent's identity.
- Let Waldo propose routines, but require explicit activation and preserve pause, revoke, evidence, and failure state.
- Make memory changes reviewable proposals with provenance and correction, not automatic team-wiki truth.
- Keep Kennel's local daemon and responsibility graph canonical; a cloud computer is an optional executor, not the owner.

**Reject**

- Replacing the user's Waldo relationship with a peer roster or faceless “agent workforce.”
- Browser universality claims.
- Automatic shared-memory admission.
- Broad credential grants or hosted custody as launch prerequisites.

## 3. Orca: Kennel work-harness reference

### What is established

**Observed:** Orca is an MIT-licensed Agent Development Environment for running CLI coding agents in isolated worktrees. Its [README](https://github.com/stablyai/orca) documents parallel worktrees, terminal splits, mobile steering, SSH worktrees, Design Mode, line-anchored diff comments, GitHub/Linear surfaces, computer use, and notifications.

High-value mechanisms include:

- A worktree lifecycle with isolated branch, files, and agent terminals; create, work, review, ship, archive, restore, and remote paths.
- A [mobile companion](https://www.onorca.dev/docs/mobile) that treats desktop as source of truth and provides a read-mostly way to monitor or steer work.
- Design context capture that sends selected DOM, computed style, a cropped screenshot, and source mapping to an agent.
- Batched line-anchored diff feedback returned to the acting agent.
- Snapshot -> targeted semantic action -> re-snapshot computer use, requiring Accessibility and Screen Recording permissions.
- SSH execution, reconnection, host verification, and remote session recovery.

**Observed security boundary:** Orca's [privacy documentation](https://www.onorca.dev/docs/telemetry) describes content-free anonymous telemetry with UI and environment opt-outs. Computer control, remote execution, mobile pairing, browser access, SSH, and skill sharing still materially enlarge the attack surface. The repository's skill-sharing threat model says shared skills are executable-author code and are not equivalent to security approval.

### Waldo decision

**Adopt**

- Worktree isolation and explicit lifecycle.
- Durable session restoration and recovery.
- Attention/unread states tied to a real next action.
- Line-anchored review returned to the correct run.
- Snapshot -> action -> verification as an interaction rhythm.

**Adapt**

- Mobile becomes a read-mostly, scoped Waldo/Kennel control surface after custody and pairing are designed.
- Design-context capture remains permissioned and source-visible.
- Provider sessions remain replaceable executors behind Kennel's daemon, authority ledger, and evidence model.
- Use mechanisms under Apache-compatible integration and attribution review; do not import its product identity.

**Reject**

- Fleet-of-coding-agents or “100x builder” positioning for Waldo.
- Provider account/rate-limit switching as a user-facing identity model.
- “Merge the winner” without verification and acceptance.
- Broad computer control without typed authority, effect receipts, and recovery.

## 4. Agent-Reach: general-task capability reference

### What is established

**Observed:** Agent-Reach is a Beta MIT Python capability layer. It describes itself as a selector, installer, health checker, and router rather than a unified wrapper. Its [channel model](https://github.com/Panniantong/Agent-Reach/blob/93ae1d18c37b707dec053c7c4f9d91cd8ef8943d/agent_reach/channels/base.py) uses ordered preferred/fallback backends and expects health/availability reporting.

Its [README](https://github.com/Panniantong/Agent-Reach) and [install guide](https://github.com/Panniantong/Agent-Reach/blob/93ae1d18c37b707dec053c7c4f9d91cd8ef8943d/docs/install.md) map web, YouTube, RSS, GitHub, X/Twitter, Reddit, Bilibili, Xiaohongshu, and other channels to upstream tools. Default installation is check-only; `--system` is the explicit authority boundary for package, configuration, skill, and MCP changes.

### Security and reliability boundary

- Login channels can depend on high-value cookies or existing browser sessions. The install guide warns about credential exposure and account bans and recommends secondary accounts.
- Installing from a mutable `main.zip`, invoking several package managers, and relying on changing upstream binaries creates supply-chain and behavioral drift.
- Private or internal URLs must not be routed through third-party readers. Retrieved web/social content is untrusted data, never an instruction source.
- Once installed, upstream tools are invoked directly. That fragments policy enforcement, provenance, auditing, rate/terms handling, and effect control.
- “Zero API fees” is not a durable capability, legality, or reliability guarantee.

### Waldo decision

**Adopt**

- A channel/capability registry.
- Explicit availability and health status.
- Ordered fallback candidates with active-backend disclosure.
- Repair prescriptions and safe default inspection.

**Adapt**

- Put every source behind a Kennel-owned capability broker rather than giving an agent direct shell entitlement.
- Declare read, draft, and effect capabilities separately.
- Pin tool versions and hashes; record source, license, terms, privacy route, and health evidence.
- Bind each invocation to purpose, permission, provenance, budget, and a receipt.
- Prefer official APIs/OAuth where available; browser-session or cookie access requires its own high-risk grant and must never be a silent fallback.

**Reject**

- “Full internet access” as a blanket grant.
- Mutable main-branch installation in production.
- Automatic browser-cookie extraction or primary-account use.
- Direct upstream calls that bypass Kennel's central authority and audit boundary.
- Integration count as a launch metric.

## Cross-signal product map

| Product layer | Instinct | Skydive | Orca | Agent-Reach | Waldo/Kennel consequence |
| --- | --- | --- | --- | --- | --- |
| Relationship | One consumer assistant | Named team agents | Coding-agent fleet | Agent-agnostic tools | Waldo stays beside the user; created agents remain coordinated through Waldo. |
| Agent creation | Public mechanics unclear | Persistent named agents and teams | Provider sessions/worktrees | Works with any command-running agent | Durable `AgentProfile`s plus temporary task agents; both are real orchestration primitives. |
| Team coordination | Public mechanics unclear | Agent teams and subagents | Parallel agent fleet | None; capability layer only | Explicit `AgentTeam`, membership, role, handoff, shared budget, and return-path contracts. |
| Presence | Text/call + claimed proactive follow-up | Web/Slack/email/iMessage/CLI | Desktop/mobile/SSH | CLI/skill | Ship desktop first; add channels without forking identity. |
| Computer | Broad assistant computer access | A cloud computer per agent + default-off local Portal | Local/remote coding environments | Upstream CLIs/browser sessions | An agent may receive an isolated local, remote, ephemeral, or durable execution environment when needed. |
| Memory | Broad persistent-context claim; public mechanics unclear | Agent-written shared Git memory | Session/worktree continuity | Config and tool health, not personal memory | Reviewable candidates and purpose-bound retrieval; no automatic truth. |
| Authority | Broad responsive actions | Scoped grants, schedules, local Portal | User steers coding sessions | Installer/tool authority | Exact typed grants, effect receipts, unknown-outcome reconciliation. |
| Proof | Public evidence contract unclear | Activity/run history | Diffs, terminals, source control | Doctor/health output | Evidence -> Verification -> user Acceptance remains the differentiator. |

## Correct Waldo platform hierarchy

The product model is not “Waldo instead of other agents.” It is “Waldo with an orchestration platform beneath and beside it.”

```text
User
  <-> Waldo agent
        - personal relationship and conversation
        - continuity across Home and Work
        - proposes teams, plans, permissions, and routines
        - stays responsible for the user return path
              |
              v
      Waldo orchestration plane
        - AgentProfile registry
        - AgentTeam membership and coordination policy
        - task-scoped AgentRuns and handoffs
        - capability, authority, budget, and credential broker
        - execution-environment allocation
        - activity, evidence, recovery, verification, and acceptance lineage
              |
              v
      Kennel execution presence
        - local daemon and canonical store
        - local worktrees, terminals, browser, and computer control
        - installed provider/runtime adapters
        - optional remote or hosted execution environments
```

### Custom agents

A durable `AgentProfile` should be user-created or explicitly saved from a successful task agent. It needs:

- name, purpose, role, and completion responsibilities;
- allowed sources, tools, skills, and data scopes;
- admitted memory scope and correction/deletion behavior;
- authority ceiling, effect policy, budget, and schedule policy;
- preferred runtime/model plus truthful fallback policy;
- execution-environment policy;
- team memberships and permitted handoffs;
- pause, revoke, export, archive, and delete controls;
- a return contract to Waldo and the owning Home/Work responsibility.

This is more than a presentation shortcut. It is a durable orchestration object. It still does not become a second Waldo identity.

### Agent teams

An `AgentTeam` should make coordination inspectable rather than hiding multi-agent work inside a transcript. Its minimum contract is:

- purpose and owning responsibility;
- member profiles and task-scoped participants;
- coordinator policy, with Waldo as the default user-facing coordinator;
- role boundaries, handoff inputs/outputs, and shared artifacts;
- capability and disclosure intersection across members;
- shared and per-agent budgets;
- stop, escalation, disagreement, and recovery policy;
- evidence aggregation and one explicit result returned to Waldo.

For a bounded run, a specialist may act as an internal team lead. It cannot widen another member's authority or replace Waldo's user-facing responsibility and return path.

### A computer for an agent when needed

The agent and its computer must remain separate objects. An `ExecutionEnvironment` may be:

- none for reasoning or API-only work;
- ephemeral local sandbox;
- durable local workspace/worktree;
- ephemeral hosted/remote sandbox; or
- durable hosted computer for an explicitly approved persistent agent.

The environment contract needs owner agent/run, filesystem and network scope, credentials route, allowed tools, resource budget, persistence class, snapshots, recovery, idle/sleep behavior, revoke, export, and deletion. Default allocation should be the smallest environment that can complete the job; persistent compute is opt-in because it adds cost, attack surface, retention, and reconciliation risk.

### How this relates to durable Waldo

[ADR 0006](../adr/0006-one-durable-waldo-multiple-governed-presences.md) already accepts one owner-scoped durable Waldo identity across desktop, mobile, voice, web, terminal, and later hosted proactive execution. That later attachment can provide always-on coordination and can broker durable computers for persistent agents.

It does **not** yet specify per-agent computer allocation. That requires a separate execution-environment specification. Hosted Waldo continuity answers “which Waldo is present and canonical?”; an agent computer answers “where does this particular run execute, what state persists, and who may access it?” Combining the two would incorrectly make compute state equal identity or responsibility truth.

## Current beta reality

The latest beta is materially ahead of the older README summary but is not a working personal agent.

- Work now has canonical **Enter**, **Understand**, and **Decide & Authorize** stages, including immutable contract/plan lineage and owner-gated approval.
- **Act & Observe** ([issue #31](https://github.com/Pin4sf/Waldo-Kennel/issues/31)), **Prove & Close** ([issue #35](https://github.com/Pin4sf/Waldo-Kennel/issues/35)), and the complete evaluation ([issue #38](https://github.com/Pin4sf/Waldo-Kennel/issues/38)) remain open.
- Home surfaces exist as truthful renderer previews, but Personal Home, Quick Capture, Open Loops, conversation, specialist runs, and insights are not canonical live behavior.
- Durable Personal Home/Open Loops ([issue #23](https://github.com/Pin4sf/Waldo-Kennel/issues/23)), Today/Catch Up ([issue #29](https://github.com/Pin4sf/Waldo-Kennel/issues/29)), shared intake ([issue #32](https://github.com/Pin4sf/Waldo-Kennel/issues/32)), and Home-to-Work lineage ([issue #40](https://github.com/Pin4sf/Waldo-Kennel/issues/40)) remain open.
- There is no live provider-neutral Waldo conversation service, capability-health broker, or governed general-task connector path.
- The pinned Node foundation failure ([issue #48](https://github.com/Pin4sf/Waldo-Kennel/issues/48)) remains a release-confidence blocker until resolved or reclassified with evidence.

## Recommended launch sequence

This is a critical path, not a feature backlog.

### Lane A — finish one provable Work loop

1. Resolve the foundation-gate failure in issue #48.
2. Implement **Act & Observe** in issue #31 with fenced Attempts, recovery, and truthful unknown-outcome handling.
3. Implement **Prove & Close** in issue #35 with evidence, independent verification labels, and user Acceptance.
4. Pass issue #38 for one locked Outcome scenario.

This creates a real action/proof engine for Waldo to coordinate. Provider completion alone must remain insufficient.

### Lane B — make Home real enough to originate responsibility

1. Persist Personal Home, explicit Quick Capture, and confirmed Open Loops through issue #23.
2. Add finite Today/Catch Up and Open Loop detail through issue #29.
3. Preserve Home-to-Work source lineage through issues #32 and #40.

Do not block this slice on ambient capture, admitted long-term Memory, Health data, mobile, broad integrations, or automatic routines.

### Lane C — wire the actual Waldo relationship

This needs a separate implementation plan and likely a dedicated issue before code work:

1. One daemon-owned conversation/episode model shared by Home Chat and the global Waldo rail.
2. One inference/runtime connection boundary using an already authenticated runtime or governed BYOK path.
3. A capability-health projection inspired by Agent-Reach: available, unavailable, degraded, active backend, permission missing, repair path.
4. A credential and permission broker: credentials outside prompt/context; read -> draft -> effect tiers; grant, revoke, expiry, and receipts.
5. Native proposal cards for every responsibility mutation or external effect.

Conversation must coordinate native objects; it must not become a second canonical database or hidden mutation path.

### Lane D — establish the orchestration-platform vertical slice

Custom agents and team coordination are core product scope, not a distant marketplace add-on. The first implementation should still be deliberately narrow:

1. Persist one user-created `AgentProfile` with purpose, capabilities, authority, memory scope, runtime connection, and execution-environment policy.
2. Create one `AgentTeam` containing Waldo, that profile, and an optional task-scoped agent.
3. Allocate an isolated local worktree/environment only to the member that needs computer execution.
4. Show the team's plan, handoffs, member state, effective permission, budget, evidence contribution, disagreement/escalation, and return path.
5. Let Waldo remain present in Home/Chat while the team operates in Activity/Work.
6. Preserve a provider-neutral seam so hosted/remote durable computers can be added after the attachment and custody gates.

This proves that Waldo is an orchestration platform without requiring unlimited teams, a marketplace, or hosted compute at the first launch.

### Lane E — launch surface and attention

Launch a private alpha as soon as one real loop passes, rather than waiting for general autonomy.

The first demo should use an authentic Waldo job:

1. The user drops several product references or a thought into Quick Capture.
2. Waldo preserves why they matter, separates fact from social urgency, and proposes one confirmed Open Loop.
3. The user connects it to a Work Outcome without losing provenance.
4. Waldo proposes a bounded plan, creates or selects the appropriate custom agents, and shows the team and exact permissions required.
5. The agent that needs a computer receives an isolated environment; the others use only their required tools.
6. Kennel shows coordination, handoffs, progress/recovery, and source-linked evidence while Waldo stays with the user.
7. Waldo returns one coherent result; the user accepts, corrects, defers, or reopens the responsibility consciously.

The attention package should contain:

- a 60–90 second end-to-end demo of that exact loop;
- a landing-page statement of the product boundary and alpha availability;
- a founder note explaining why “an agent that acts” still leaves users carrying continuity, permission, proof, and closure;
- a small design-partner cohort selected around recurring unfinished personal/work responsibilities;
- a public build log showing real outcomes and failures without fabricating autonomy or traction.

Suggested positioning:

> Waldo is one personal agent across your life and work—and the place where you create and coordinate the agents that help it. Kennel gives those agents governed tools and computers, and lets you see what they remember, propose, do, hand off, and prove.

### What to defer until after the first loop

- Ambient screen/audio capture in issue #36.
- Durable Memory admission beyond correctable candidates in issues #20 and #39.
- An open agent marketplace, public agent sharing, and unlimited team topology. User-created agents and one bounded team remain part of the core vertical slice.
- Always-on durable computers for every agent. Allocate compute when needed; hosted persistence follows the attachment, custody, and cost gates.
- Mobile or hosted attachment in issues #24 and #28.
- Automatic routines beyond explicit proposal/activation.
- Broad social/web connector parity or direct Agent-Reach installation.
- Multi-provider routing whose main value is provider switching rather than user outcomes.

## Launch gates and learning metrics

The alpha is ready to invite users when:

1. Canonical mode shows no preview conversation, responsibility, run, evidence, or outcome as live.
2. One captured thought can become a user-confirmed Open Loop and retain provenance.
3. One Open Loop can link to one Work Outcome without moving or merging either lifecycle.
4. One approved plan can run under a frozen authority envelope and recover from interruption.
5. The user can see what is working, waiting, failed, unknown, evidenced, verified, and accepted.
6. Credential material never enters renderer state, transcripts, prompts, logs, or committed files.
7. Revocation and narrowed authority fail closed.
8. The same Waldo relationship resumes with correction and source lineage intact.
9. The user can create, inspect, pause, and revoke one durable custom agent.
10. Waldo can coordinate one bounded multi-agent team and return one coherent result with member-level evidence.
11. An agent receives an isolated computer only when required, and its persistence, scope, cost, revoke, recovery, and deletion state are visible.

Measure the loop, not attention vanity:

- time from capture to a confirmed responsibility;
- percentage of accepted runs with inspectable evidence;
- approval edits, denials, corrections, and revocations;
- unknown outcomes successfully reconciled;
- consciously closed, deferred, released, or reopened responsibilities;
- users who return with a second real responsibility after the first completed loop.

Do not claim traction, reliability, memory quality, or autonomy from waitlist size, social impressions, connector count, or provider completion.

## Decision

These references and the product clarification **expand the explicit platform contract without replacing the existing responsibility architecture**. They pull five capabilities forward within the first working slice:

1. capability health and truthful degradation;
2. explicit grant/revoke plus credential isolation before real effects; and
3. durable user-created `AgentProfile`s;
4. one inspectable `AgentTeam` with need-based isolated compute; and
5. one end-to-end evidence-bearing launch story in which Waldo stays beside the user while the team works.

The correct competitive response is focused platform speed: finish the first real Waldo loop, prove custom-agent/team orchestration inside it, make that visually undeniable, invite users into a bounded alpha, and learn from completed responsibilities. Waldo should grow toward capability parity with the useful primitives these products expose, but through one coherent architecture rather than copying their agent count, workbench breadth, or channel count as disconnected features.
