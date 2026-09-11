# Claude Code reasoning adapter gap

Status: frontend exposes the limitation truthfully; backend support is not implemented.

## Current boundary

Claude Code is an admitted execution harness and the existing
`backend/internal/adapters/chatdriver/claudeacp` adapter is a `ports.ChatDriver`.
It can discover the user-owned Claude binary, use the packaged ACP runtime, and
preserve the user's local login/subscription. That adapter does not implement
the provider-neutral `ports.LLMClient` used by Waldo's `IntelligenceProvider`,
and it does not expose a proven constrained intelligence boundary.

The daemon's reasoning settings currently support `anthropic` (direct Claude
API), `openai` (direct OpenAI API), and `codex` (Codex App Server). The
`claude-code` execution identity must therefore not be persisted as a ready
reasoning provider or silently mapped to the direct Anthropic API.

## Required backend contract before enabling Claude Code reasoning

1. Add a provider-neutral intelligence adapter that translates bounded Contract
   and Plan requests into Claude Code ACP turns and returns structured
   `ports.LLMClient` responses with honest effective-provider/model provenance.
2. Reuse the existing Claude plugin boundary for `ResolveBinary` and
   `AuthStatus`. Authentication must remain the user's local Claude Code
   subscription/login; Kennel must not collect, copy, or persist a Claude
   credential as an API key.
3. Prove the intelligence role boundary before reporting readiness: packaged
   ACP runtime and Claude binary resolution, authenticated account state, model
   selection semantics, structured-output support, and a no-tool/constrained-
   read mode that cannot create execution authority or external effects.
   Ordinary Claude chat capability is not sufficient evidence.
4. Add a local/protocol-only availability probe analogous to Codex
   `ProbeIntelligence`. It may report that the adapter can be attempted, but it
   must not mark a model verified or make a billed/model turn.
5. Extend owner-triggered verification through the existing settings
   verification path with a minimal structured probe. Verification must remain
   explicit, provider/model-bound, and separate from Outcome acceptance.
6. Extend the daemon settings/API schema and tests for the explicit
   `claude-code` reasoning binding, including unavailable/auth-required/
   capability-unproven states. Do not add a fallback chain.

Until those seams exist and are verified, onboarding may select Claude Code for
execution sessions, but it must explain that Waldo reasoning through Claude
Code is unavailable and direct the owner to choose Codex App Server sign-in or
an API provider in Settings.
