# Kennel

Kennel is an outcome-first desktop workspace for directing coding agents.

Most agent tools begin with a chat or a task. Kennel begins with the result you want. You describe an **Outcome**, clarify what success means, review the proposed deliverables and objective checks, and approve the plan before implementation starts. Kennel then coordinates the available coding-agent harnesses, keeps their work legible, and returns the decisions and evidence needed to judge whether the Outcome was actually delivered.

Kennel was built as an abstraction on Agent Orchestrator, retaining its local execution, agent-harness, workspace, and source-control foundations while moving the product toward an Outcome-driven workflow.

## The central idea

The Outcome is the unit of truth.

An agent session is execution activity beneath an Outcome. A completed model turn, an idle terminal, a passing check, or even a merged pull request can be useful evidence, but none of those events independently proves that the user's intended result has been achieved. Kennel keeps intent, execution, evidence, review, and human acceptance separate so that work can be supervised without surrendering judgment to the agents performing it.

## How Kennel works

1. **Understand the project**

   When a project is opened, the selected orchestrator performs a quick, read-only orientation of the codebase. Kennel can use that context to suggest useful Outcomes before the user writes one.

2. **Define an Outcome**

   The user describes the result they want rather than manually decomposing it into implementation tasks.

3. **Resolve ambiguity**

   If material details are missing, the orchestrator asks focused clarifying questions. Kennel presents relevant choices and always allows a custom answer.

4. **Review the orchestration plan**

   Once the Outcome is clear, Kennel presents a reviewable plan containing:

   - concrete deliverables;
   - objective completion checks;
   - constraints and exclusions;
   - the agent assigned to each part of the work;
   - a graph showing the orchestrator, worker agents, their responsibilities, and how they converge on the Outcome.

5. **Approve before execution**

   The user can approve the plan or continue refining it. The orchestrator must not begin implementation or spawn implementation workers until the plan is explicitly approved.

6. **Delegate bounded work**

   After approval, the orchestrator breaks the plan into focused work and delegates it to installed harnesses such as Codex, Claude Code, or OpenCode. Workers operate in isolated workspaces where appropriate, while the orchestrator coordinates dependencies, blockers, review feedback, and verification.

7. **Follow progress on the board**

   The Kanban board is the default operational view. Work moves through meaningful states:

   - **Working** while implementation is active;
   - **Needs You** when a decision, answer, failed check, or requested change requires attention;
   - **In Review** while deliverables and checks are being validated;
   - **Ready to Merge** when the approved work is complete and supported by evidence.

   Provider idleness is not treated as completion. Workers and orchestrators report explicit lifecycle progress, while source-control, CI, and review facts remain authoritative when available.

8. **Inspect and decide**

   The user can open the orchestrator conversation at any time, inspect individual workers, review changes and evidence, revise the plan, or provide a blocked decision. Kennel prepares the work for acceptance; the human retains authority over merging and whether the Outcome is truly complete.

## Product principles

- **Outcomes before tasks.** Users define the result and its success conditions; Kennel owns the decomposition.
- **Plans are contracts.** Deliverables and checks are visible and editable before agents receive write authority.
- **Orchestrators coordinate; workers implement.** Planning, delegation, implementation, and review have distinct responsibilities.
- **Evidence over claims.** Completion should be supported by changed files, tests, checks, reviews, and explicit receipts—not an agent saying it is finished.
- **Parallel work stays legible.** Each worker has a bounded responsibility, visible ownership, and an isolated execution context where needed.
- **Human judgment remains final.** Kennel can recommend, verify, and prepare work, but it does not silently widen authority or accept an Outcome for the user.
- **Local custody by default.** Projects, agent activity, and orchestration state are managed locally, with provider access constrained to the work the user placed in scope.

## What Kennel includes

Kennel is an Electron desktop application with a local backend service. Its current foundation includes:

- project-aware orchestrator sessions;
- multiple coding-agent harnesses and models;
- structured chat and native terminal interfaces;
- Git branches and isolated worktrees for parallel workers;
- an Outcome clarification and plan-approval workflow;
- deliverable, completion-check, and agent-assignment views;
- a live Kanban derived from agent, pull-request, CI, and review state;
- direct access to orchestrator and worker conversations;
- pull-request, review, preview, and browser-assisted development surfaces.

The goal is not to become another agent chat window. Kennel is intended to be a trustworthy local operating environment for turning human intent into coordinated, reviewable, evidence-backed agent work.

## Development

The repository contains the Electron frontend, Go backend, shared product UI, provider adapters, and packaging scripts. Generated runtimes and compiled binaries are intentionally excluded from Git and are rebuilt by the development scripts.

Start with [the development guide](docs/development.md) for prerequisites, local setup, build commands, and verification. The main checks are:

```sh
npm --prefix frontend test
npm run frontend:typecheck
npm run product-ui:check
npm run lint
```

## License

See [LICENSE](LICENSE).
