# ADR 0012 — Waldo reasons with the owner's model, and has no deterministic floor

- **Status:** Accepted
- **Date:** 2026-09-08
- **Amends:** ADR 0003 (Local-first Waldo Core)
- **Depends on:** ADR 0011 (Go control plane and non-authoritative intelligence)

## Context

Kennel shipped with no model of its own. "Waldo thinking" meant spawning the
owner's Codex CLI, handing it a prompt, and asking it to `curl` structured JSON
back to loopback within fifteen minutes. Behind that sat a rule-based analyzer:
a regex for the word "today", a title made from the first eight words of the
statement, and one fixed success criterion.

Both halves failed in the way that matters. In a full UX audit against `beta`,
the spawned agent never answered, the request expired, and **no Outcome was
ever created** — the user typed a goal and got an empty list with no error and
no retry. The rule-based floor existed to prevent failing closed, and its
actual effect was to make a broken front door look like a working one.

The floor also misrepresented the product. Kennel's claim is that it turns
intent into a precise, verifiable Contract. A canned criterion is not
understanding, and shipping one teaches users to distrust everything above it.

## Decision

Waldo reasons through a real model, and there is nothing behind it.

1. `ports.LLMClient` is Waldo's own reasoning surface, provider-neutral and
   deliberately separate from the agent adapters that execute authorized work.
   Coding agents execute; Waldo thinks; neither borrows the other's authority.
2. `adapters/llm/anthropic` is the first implementation, using structured
   outputs so a reply is schema-constrained by the provider and revalidated by
   Kennel.
3. `intelligence.LLMProvider` implements the existing `IntelligenceProvider`
   port for both Contract analysis and Plan drafting. No new seam is invented.
4. The deterministic floor is **deleted**, not retained as a fallback:
   `intelligence.DeterministicProvider`, `intake.RuleBasedAnalyzer`, the
   daemon's agent-spawn intake analyzer, and the agent-spawn decomposition
   proposer are all removed.
5. With no reasoning credential configured, intake fails **retryably and says
   why**. An honest failure is better than a canned proposal.
6. The credential is the owner's. It is read from `KENNEL_WALDO_API_KEY`, or
   `ANTHROPIC_API_KEY` as a fallback, and never leaves the machine.

## Relationship to ADR 0003

ADR 0003 says launch does not require "a Waldo-operated LLM API or Waldo-funded
inference", and that "planning, execution, critique, and summarization use the
user's authenticated provider sessions".

This decision honors both. Waldo is not operating or funding inference: the key
belongs to the owner, billing is theirs, and the call is made from their
machine. What this amends is the narrow reading that "the user's authenticated
provider session" meant only a logged-in coding CLI. A user-held API key is the
same custody with better reliability, and reliability is the whole point —
Waldo's own reasoning needs a call it controls, not a coding agent that may or
may not decide to answer.

Local-first is unchanged. No account, no hosted Waldo, no Waldo-funded call.

## Consequences

### Benefits

- the front door works: Contract proposals arrive synchronously, or fail with a
  reason the owner can act on;
- Waldo's voice becomes usable for what it was always specified to do — drafting
  contracts, drafting plans, summarizing receipts, answering the rail;
- one honest failure mode replaces two silent ones;
- the control plane is untouched. Models propose; Go still owns authority,
  graph shape, routing, capability ceilings, idempotency, fencing, and
  acceptance, exactly as ADR 0011 requires.

### Costs

- Kennel now has a hard runtime dependency on a configured reasoning key, and
  an unconfigured install cannot create an Outcome. This is deliberate and must
  be surfaced in onboarding rather than hidden behind a fallback;
- inference costs the owner money on a path that was previously free;
- composed-Outcome decomposition has no proposer until it is model-backed.
  `AskForDecomposition` fails closed and says so. Hand-authored decomposition
  is unaffected — it never went through a proposer;
- key storage is currently environment-only. A settings surface, and secret
  storage under `~/.kennel`, remain to be built.

## Non-goals

- no Waldo-hosted inference, and no Waldo-funded fallback model;
- no second reasoning path that re-introduces a floor;
- no model involvement in authority, acceptance, dependency validation,
  idempotency, recovery fencing, effect reconciliation, or evidence binding.
