# ADR 0013 — Add an explicit Codex harness mode for Waldo reasoning

- **Status:** Accepted
- **Date:** 2026-09-10
- **Amends:** ADR 0012 (Waldo reasons with the owner's model)
- **Depends on:** ADR 0011 (Go control plane and non-authoritative intelligence)

## Decision

ADR 0012's direct Anthropic and OpenAI API modes remain supported. This narrow
amendment adds a third, explicit reasoning selection: `codex`.

1. `codex` means the owner has selected the installed, signed-in Codex app
   server as Waldo's reasoning harness. Kennel does not extract Codex auth
   tokens or turn the harness into a generic API client.
2. The existing `ports.LLMClient` and `intelligence.LLMProvider` seams remain
   unchanged. The Codex adapter uses the installed app-server protocol's
   native `turn/start.outputSchema` field and returns only the settled assistant
   JSON to the existing Go parser and domain validators.
3. Planning identity is scoped to the proposal request. Every call opens a
   fresh ephemeral Codex thread, records its native thread id only as
   provenance, and creates no Kennel `AgentSessionRef`, Attempt, WorkUnit, or
   durable Outcome conversation.
4. The thread receives the already-approved repository/document grounding
   packet in the prompt. Its cwd is a disposable temporary directory. The
   adapter requests `approvalPolicy=never`, `sandbox=read-only`, and a
   turn-level `networkAccess=false` policy, and requests that plugins, apps,
   skill instructions, and MCP servers be absent. These requests are not yet a
   sufficient confinement proof: the installed app-server's MCP session layer
   is additive, and read-only does not establish an approved-packet-only read
   root. Production Codex reasoning therefore fails closed as unavailable until
   a deterministic no-tool or constrained-read boundary is proven. Controlled
   protocol tests may exercise the adapter behind that capability gate.
5. Model and effort are explicit settings. An empty value preserves Codex's
   provider-default semantics. Kennel never silently substitutes a model or
   another reasoning provider. A failed or unavailable Codex harness is an
   actionable reasoning failure.
6. Codex app-server probe evidence establishes only local availability,
   protocol compatibility, model-list reachability, and explicit auth failure
   when reported. It does not set `Verified`. Verification remains an
   owner-triggered, subscription-backed structured call.
7. Cancellation interrupts the named provider turn and closes the ephemeral
   conversation. Stale, replayed, malformed, incomplete, or non-completed
   responses are rejected; none can become canonical state. The Go control
   plane remains the authority for proposal validation, revision binding,
   idempotency, capability checks, approved checks, and acceptance.

## Consequences

- The settings API exposes `mode=codex_harness`, `provider=codex`, and truthful
  local readiness without pretending that a Codex sign-in is an API key.
- The bounded adapter and settings path are present, but the installed Codex
  harness is truthfully unavailable for Waldo proposals until its confinement
  boundary is proven. This is preferable to treating prompt instructions or a
  disposable cwd as a security boundary.
- This slice does not define the later worker-side skill/tool bridge
  (`understand-and-propose`, `execute-work-unit`, `report-result`, or
  `request-help`), nor does it make a proposal thread durable. Those require a
  separate bounded capability contract and acceptance evidence.
- Provider subscription execution remains a live-conformance concern. Controlled
  peer tests prove the daemon/adapter protocol boundary; they do not prove that
  the owner's installed Codex account can complete a real planning turn.

## Non-goals

- no generic Codex API-token extraction;
- no hidden fallback between direct APIs and Codex;
- no worker launch, approval, external effect, or Outcome acceptance authority;
- no claim that a readiness probe is a successful planning or production run.
