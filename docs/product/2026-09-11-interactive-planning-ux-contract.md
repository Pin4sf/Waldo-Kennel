# Interactive planning UX contract

The simplest useful shape is one inline step inside Mission Control:

```text
Confirmed Outcome
      ↓
Choose planner + Read this repository
      ↓
Short planning conversation
      ↓
Review Plan
      ↓ owner approval
Serial WorkUnit supervision
```

## What the owner sees

- A single planner picker. Ready choices are selectable; unavailable choices explain the one next action. There is no automatic fallback.
- One default context choice: **Read this repository**. An optional supplied-document packet appears only when one is already approved.
- A familiar compact composer. The planner's ordinary text stays brief.
- Clarifications as a question with a recommended answer and a few alternatives.
- Contract-change suggestions as **Review Contract change**. Accepting the suggestion opens the existing Contract editor; it never edits or reconfirms behind the owner's back.
- A Plan card only after a valid structured proposal is compiled. WorkUnits, dependencies, provider/model binding, authority, checks, assumptions, and blockers remain reviewable before **Approve Plan**.

## State mapping

| Daemon fact | UI treatment |
|---|---|
| no ready candidate | compact setup state; no Start button |
| `active` + `waitingOn=owner` | composer enabled |
| `active` + `waitingOn=provider` | one in-place thinking indicator; composer disabled |
| clarification turn | short question card |
| Contract-change proposal turn | review action into existing Contract editor |
| `proposal_ready` | Plan review is primary; conversation remains readable |
| `superseded` | explain Contract changed; one **Start fresh** action |
| `cancelled` | read-only history with **Start new planning** |

## Progressive disclosure

The main surface shows Outcome, planner, latest exchange, and Plan summary. Repository digest, IntelligenceRun identity, exact routing rationale, grants, checks, and full WorkUnit details stay available in disclosure rows. Provider transcript or terminal is never required for ordinary planning.

## Safety language

Use one concise disclosure near context selection:

> The planner can read a bounded snapshot of this repository. It cannot run commands, edit files, or make external changes while planning.

Plan approval and execution remain visibly separate actions. A generated Plan is ready for review, not accepted and not running.
