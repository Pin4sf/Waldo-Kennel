# Waldo Kennel Positioning and Competitive Thesis

- **Status:** Product positioning authority for the first Outcome-control-plane MVP
- **Locked:** 2026-09-08
- **Implementation companion:** `docs/superpowers/plans/2026-09-08-outcome-control-plane-mvp-reset.md`
- **Architecture companions:** ADR 0010 and ADR 0011

## 1. Category

Waldo Kennel is an **Outcome control plane for agentic software work**.

It is not primarily:

- another coding agent;
- a terminal multiplexer for agents;
- a worktree manager;
- a kanban board for sessions;
- a model router;
- a wrapper that simply starts several agents at once.

Those are useful implementation capabilities and several strong products already provide them.

Kennel's product object is the **Outcome**: a durable desired state with an explicit Contract, execution authority, Plan, WorkUnits, Attempt lineage, evidence, verification, and owner acceptance.

The shortest positioning line is:

> **Stop managing agent sessions. Manage outcomes.**

A more complete version is:

> **Waldo Kennel turns a desired outcome into a durable, governed execution contract that can be carried across agents, models, sessions, retries, and restarts until the result is proven and accepted.**

## 2. The problem we are actually solving

Agent capability is becoming abundant. Coordination is becoming the bottleneck.

A user can already ask Codex, Claude Code, Cursor, Devin, Factory, OpenCode, or another agent to implement a task. The difficult part appears once the work is non-trivial or spans more than one run:

- What exactly did we agree the result should be?
- Which assumptions and constraints are still authoritative?
- What did the user approve the system to do?
- Why was this provider/model selected?
- Which work has really completed versus merely stopped running?
- Which run produced which evidence?
- What changed after a retry or handoff?
- What still remains before the result is actually acceptable?
- Can another agent continue without reconstructing the whole story from transcripts?

Most agent products are optimized around **sessions, tasks, workspaces, branches, or agents**. Kennel is optimized around the **state of the intended outcome**.

That distinction matters only if the product maintains it end to end. If Kennel eventually collapses back into a session dashboard, it has no meaningful USP.

## 3. The USP

The first defensible wedge is the combination of five properties, not any one feature by itself.

### 3.1 Outcome is the durable identity

A provider session can die, be replaced, be retried, or hand work to another provider without replacing the responsibility being supervised.

Canonical lineage:

```text
Outcome
→ ContractRevision
→ PlanRevision
→ WorkUnit
→ Attempt
→ AgentSessionRef
→ EvidenceItem
→ VerificationRun
→ AcceptanceDecision
```

The user should be able to answer "what is true, what remains, and why?" without reading every provider transcript.

### 3.2 Intelligence is separated from authority

Models may analyze the project, propose a Contract, draft a Plan, or recommend a provider/model. None of those actions authorizes execution.

The owner explicitly crosses the authority boundary by approving the Plan.

That lets Kennel use increasingly capable intelligence without making model output itself the control plane.

### 3.3 Execution binding is explicit and survives mutable preferences

Project and Outcome preferences help routing. They do not silently change already-approved work.

Once a WorkUnit is approved, its provider/model semantics are frozen. An Attempt consumes that exact binding rather than re-reading mutable Project settings or applying a hidden fallback.

### 3.4 Completion is evidence-based, not session-based

"Agent finished" is not equivalent to "Outcome achieved."

Work is associated with evidence, verification, and an explicit owner AcceptanceDecision. Provider completion is one observation among several, not the final authority.

### 3.5 Cross-agent continuity is a control-plane property

The Outcome's Contract, Plan, decisions, bindings, evidence, and current state belong to Kennel rather than to one provider's hidden chat context.

This is the foundation for switching agents without rebuilding the work from scratch and, later, for Waldo to supervise work on the user's behalf.

## 4. Why someone should use Kennel

Kennel should be used when **the coordination cost of agentic work is becoming material**.

The strongest first users are developers and technical builders who:

- regularly run multiple coding-agent sessions;
- use more than one harness/model;
- work on tasks too large for one clean prompt-to-PR run;
- lose time reconstructing what previous runs decided;
- need to review, retry, redirect, or hand off work;
- care about knowing why something is considered complete;
- want agents to do more execution without surrendering the approval boundary.

The practical promise is:

> Tell Kennel what should become true. Review the Contract. Approve the Plan. Then supervise the Outcome while Kennel coordinates the agent runs underneath it.

### When Kennel is unnecessary

We should be candid about this.

If a user has one small coding task, prefers one agent, and is satisfied with one prompt → one branch/PR, an additional control plane is overhead. Cursor, Claude Code, Codex, Devin, Factory, or another single-agent workflow may be the better product.

Kennel earns its place when the user would otherwise begin manually coordinating **multiple runs, decisions, branches, approvals, retries, and proofs**.

## 5. Competitive landscape

This market already contains excellent products. Our USP cannot rely on claims they have already commoditized.

### 5.1 Integrated coding agents

**Cursor**

Current Cloud Agents run long-lived coding work in isolated VMs, can operate in parallel, produce proof artifacts such as videos/screenshots/logs, and can be launched from several surfaces. Cursor also supports async subagents and worktrees.

What this invalidates as a Kennel USP:

- background execution;
- isolated work;
- agent artifacts;
- subagent parallelism;
- one interface for several runs.

Reference: https://cursor.com and Cursor Cloud Agent documentation.

**Devin**

Devin can now manage multiple Devins in parallel. A coordinator Devin scopes work, launches managed Devins on isolated VMs, monitors progress, resolves conflicts, and compiles results. Devin also maintains knowledge/playbooks and session analysis.

What this invalidates as a Kennel USP:

- manager/worker agent topology;
- automatic decomposition into parallel agents;
- session analytics;
- persistent team knowledge by itself.

Reference: https://docs.devin.ai/work-with-devin/advanced-capabilities

**Factory**

Factory Droids plan, write, test, and ship code across several interfaces. Factory advertises adjustable autonomy and model routing across Claude/GPT/Gemini, and is expanding toward an end-to-end software-factory system.

What this invalidates as a Kennel USP:

- model routing;
- adjustable autonomy as a generic feature;
- one-prompt-to-PR execution;
- multi-surface agent access.

Reference: https://factory.ai/product/droids

### 5.2 Multi-harness agent orchestration/workspace products — closest direct competitors

**Warp Oz**

Oz is explicitly positioned as a cloud agent orchestration platform and multi-harness control plane. It launches and governs cloud agents, supports parallel orchestration, automation, audit trails, and cross-harness memory.

This is one of the closest competitive threats because "multi-harness control plane" is already their language.

Kennel therefore must differentiate at the **Outcome/authority/evidence object model**, not at "we can orchestrate Claude and Codex."

Reference: https://www.warp.dev/blog/multi-harness-cloud-agent-orchestration

**Superset**

Superset gives tasks isolated worktrees, supports many coding harnesses, parallel workstreams, agent-driven orchestration, automations, review and PR flows.

Its core unit is a workspace/task and the agents working inside those workspaces.

Reference: https://docs.superset.sh/

**Conductor**

Conductor provides a workspace per task with its own worktree, branch, chat, terminal, setup/run scripts, diff, checks, and PR flow. It is strong at local parallel agent ergonomics.

Reference: https://www.conductor.build/docs/guides/parallel-agents/run-multiple-codex-sessions

**Nimbalyst**

Nimbalyst provides a visual multi-agent kanban, parallel Claude Code/Codex sessions, workstreams, session chaining, worktree isolation, and live monitoring.

Reference: https://nimbalyst.com/features/agent-orchestration/

**Agent Orchestrator (AO)**

AO manages fleets of coding agents in separate worktrees/branches/PRs, including CI/review follow-up, with agent/runtime/tracker abstraction.

Kennel originated with this kind of orchestration substrate, but the new product should not inherit the session/worktree ontology as its top-level product model.

Reference: https://github.com/useagent/ao

### 5.3 Personal agents — strategically adjacent to future Waldo

**Poke**

Poke lives in messaging surfaces and manages email, calendars, reminders, web search, and integrations.

Reference: https://poke.com/docs

**folk**

folk positions itself as a personal AI in messaging that acts, remembers, runs scheduled/autonomous tasks, manages email, researches, books, and can even code from the phone.

Reference: https://www.folk.com/use-cases

**Tomo**

Tomo is personal AI in texts that learns goals, manages life context, proactively messages, and connects to email/calendar.

Reference: https://www.tomo.ai/about

**OpenInstinct**

OpenInstinct is an open-source/self-hostable personal iMessage assistant with browser action, private credentials/context, and model flexibility.

Reference: https://github.com/Merit-Systems/OpenInstinct

These products validate the longer-term personal-agent thesis: users want a persistent relationship that can act across their life. They are not, however, the first Kennel wedge. Kennel begins with the harder-to-hide coordination problem in agentic software work and can later become the work execution surface behind Waldo.

## 6. Competitive map

| Product class | Primary durable object | Strongest value | What Kennel must not claim as unique | Kennel differentiation target |
| --- | --- | --- | --- | --- |
| Cursor / Devin / Factory | Agent task/session/run | Very capable integrated execution | autonomy, parallel subagents, proof artifacts, model routing | provider-independent Outcome authority + continuity |
| Superset / Conductor / Nimbalyst / AO | Workspace/task/session | Parallel harness/worktree supervision | multi-agent dashboard, worktrees, terminal/diff/PR UX | Outcome/Contract/Plan/Evidence lifecycle above sessions |
| Warp Oz | Agent/job + orchestration control plane | Cloud multi-harness scale/governance | "multi-harness control plane", orchestration, memory | user-owned Outcome contract + explicit authority + acceptance |
| Poke / folk / Tomo / OpenInstinct | User relationship/conversation | Persistent personal action | persistent assistant, integrations, proactive tasks | future Waldo relationship connected to governed execution |
| Waldo Kennel | Outcome | Durable governed execution across agents | — | Contract → authority → execution → evidence → acceptance |

## 7. What the user should feel

Bad mental model:

> "I have five agent terminals open. Which one is doing what?"

Target mental model:

> "I have three Outcomes. One needs my decision, one is executing, and one is ready for me to review. I can inspect the agent runs if I need to."

The difference is subtle in a feature list and enormous in product behavior.

The default Kennel UI must therefore show responsibility and consequence before process details.

## 8. The wedge and the larger Waldo thesis

Kennel should not try to become the entire Waldo personal-agent thesis in the first release.

The near-term wedge is:

> **People using many coding agents are acquiring coordination debt faster than agent capability is improving. Kennel owns the durable Outcome above those agents.**

The longer-term convergence is:

```text
THE PERSON
identity • memory • goals • commitments • authority
             │
             ▼
           WALDO
one durable user-owned relationship
             │
             ▼
         OUTCOMES
contract • authority • continuity
             │
             ▼
     KENNEL CONTROL PLANE
plan • routing • execution • proof
             │
             ▼
   SPECIALIST AGENTS / TOOLS
Codex • Claude • Cursor • etc.
```

Kennel is therefore not a separate philosophical bet from Waldo. It is the first concrete infrastructure layer where the "one durable relationship, many specialist intelligences" thesis can be made operational and falsifiable.

## 9. Defensibility

No individual UI feature here is a moat. Competitors can copy boards, graphs, approvals, and routing controls.

Potential defensibility comes from compounding state and workflow:

1. **Outcome ledger** — durable history of contracts, decisions, plans, attempts, evidence, verification, and acceptance.
2. **Cross-harness continuity** — execution can move without losing canonical intent/authority.
3. **Outcome-level evaluation data** — which decomposition/routing/recovery strategies actually produce accepted results, not merely successful agent sessions.
4. **User authority model** — a reusable representation of what the user allows systems to decide/do.
5. **Project continuity** — accumulated proven context and decisions that improve future Outcomes.
6. **Future Waldo continuity** — the same governed Outcome model can be initiated and supervised by the user's personal agent.

The strongest long-term data asset is not "agent transcripts." It is structured information about **what users wanted, what was authorized, how systems tried to achieve it, what evidence was produced, and what users ultimately accepted**.

## 10. MVP falsification tests for the USP

The MVP does not prove this positioning merely by implementing the domain types.

We should consider the thesis supported only if dogfood users can demonstrate the following:

- after two or more agent Attempts, they can understand current Outcome state without reconstructing transcripts;
- changing provider/model between WorkUnits does not destroy continuity;
- a failed/retried Attempt does not rewrite or confuse prior execution history;
- the system can explain why work is blocked or needs approval;
- "ready for review" is supported by evidence rather than inferred from session termination;
- the owner can reject/continue work after verification without manually reconstructing the task;
- supervision time is materially lower than coordinating the same work manually across agent sessions.

If users still spend most of their time opening individual sessions to understand what happened, the Outcome abstraction has failed regardless of how elegant the backend is.

## 11. Positioning language to use

### Primary

> **Stop managing agent sessions. Manage outcomes.**

### Product description

> Waldo Kennel is an Outcome control plane for agentic software work. Define what should become true, approve how agents are allowed to pursue it, and supervise the result across models, sessions, retries, and handoffs until it is proven and accepted.

### Short explanation

> Coding agents are getting good enough that running them is no longer the hard part. The hard part is carrying intent, decisions, authority, progress, and proof across all of the runs. Kennel keeps that durable state above the agents.

## 12. Claims we should avoid

Do not position Kennel primarily as:

- "the first multi-agent coding platform";
- "the first multi-harness control plane";
- "the easiest way to run agents in parallel";
- "automatic model routing";
- "persistent memory for coding agents";
- "a better agent kanban";
- "one agent to manage all agents" without explaining the governed Outcome model.

All of these are either already occupied claims or too easy to converge on.

The product has a reason to exist only if this remains true:

> **Agents are workers. Sessions are execution detail. The durable thing the user owns is the Outcome.**
