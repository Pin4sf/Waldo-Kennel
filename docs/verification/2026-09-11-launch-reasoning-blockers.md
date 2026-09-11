# Packaged launch reasoning blockers

## Current update

The credential/testing restriction described below is historical and superseded: the user authorized bounded live tests. The existing OpenAI credential was verified with `gpt-5-nano` and `minimal` effort; daemon verification persistence was repaired at `92aa0f14b`. Project import is working for the actual `agent-orchestrator` repository. A read-only Outcome was created there through the desktop as `out-fddc4613-1a03-4951-9d1e-c6020033dc5a`, with network and PR permissions explicitly switched off before confirmation.

See [the usability iteration](2026-09-11-launch-usability-iteration.md) for current provider test evidence and remaining blockers. Codex session creation/streaming/reopen passed; bounded Codex reasoning remains unproven. Claude resume standing-context testing failed. OpenCode live testing is not established. Plan generation remains blocked by forbidden shell checks.

The real-project exercise also exposed a bounded context bug: exhausting the 512-entry discovery budget discarded previously discovered root files. The repair retains safely inspected candidates, explicitly marks partial context, and includes both the facts and the limitation in the model request. Main documents have priority; long files contribute explicitly marked bounded prefixes. A regression test reproduces the previous empty context and passes with the repair. A read-only probe of the actual imported repository now includes AGENTS.md, README.md, package.json, docs/STATUS.md and docs/architecture.md, with the three long documents marked truncated and the discovery limit disclosed. Intelligence/outcome package tests and intelligence race tests pass. This does not claim comprehensive repository inspection or a completed provider execution.

## Historical checkpoint

Observed on candidate 02d35d1ec through native computer use, isolated daemon port65025.

Onboarding reports Codex signed in and binds it for reasoning. In Scratch, submitting a bounded local file Outcome saves the intake but returns INTAKE_ANALYSIS_FAILED. Configure reasoning reports no proven no-tool or constrained-read boundary. No Outcome is created.

Source: codexappserver.Driver.intelligenceBoundaryAvailable defaults false in New. Both ProbeIntelligence and startIntelligence reject this state. Production does not establish the boundary. Therefore the existing adapter code and sign-in are not evidence that Codex reasoning is available. The frontend sign-in-only repair advice was incorrect; this checkpoint changes it to retain the daemon reason and distinguish capability from authentication.

Claude backend handoff: prove and implement constrained Codex reasoning and the missing Claude Code intelligence adapter using installed protocol capabilities and provider conformance. Preserve frozen policy, explicit choice, structured output and no hidden fallback. Do not simply set intelligenceBoundaryAvailable=true. Both harnesses need actual protocol-boundary tests and a real packaged reasoning attempt.

An explicit OpenAI API option is visible with secure key input. No credential was entered and no direct API verification was attempted in this resumed audit. Prior automatic approval review rejected billed credential-backed verification; owner action/authorization remains required.

Project import regression remains incomplete: native chooser held Open disabled for an existing disposable Git folder; cancel and return to Work succeeded. Current branch source fix has targeted tests but no successful live registration claim.

No push, merge to beta, deploy, release or owner acceptance.
