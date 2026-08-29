package domain

// AgentHarness identifies which agent CLI/runtime a session drives.
type AgentHarness string

// Supported agent harnesses.
const (
	HarnessClaudeCode AgentHarness = "claude-code"
	HarnessCodex      AgentHarness = "codex"
	HarnessAider      AgentHarness = "aider"
	HarnessOpenCode   AgentHarness = "opencode"
	HarnessGrok       AgentHarness = "grok"
	HarnessDroid      AgentHarness = "droid"
	HarnessAmp        AgentHarness = "amp"
	HarnessAgy        AgentHarness = "agy"
	HarnessCrush      AgentHarness = "crush"
	HarnessCursor     AgentHarness = "cursor"
	HarnessQwen       AgentHarness = "qwen"
	HarnessCopilot    AgentHarness = "copilot"
	HarnessGoose      AgentHarness = "goose"
	HarnessAuggie     AgentHarness = "auggie"
	HarnessContinue   AgentHarness = "continue"
	HarnessDevin      AgentHarness = "devin"
	HarnessCline      AgentHarness = "cline"
	HarnessKimi       AgentHarness = "kimi"
	HarnessMuse       AgentHarness = "muse"
	HarnessKiro       AgentHarness = "kiro"
	HarnessKilocode   AgentHarness = "kilocode"
	HarnessVibe       AgentHarness = "vibe"
	HarnessPi         AgentHarness = "pi"
	HarnessKimchi     AgentHarness = "kimchi"
	HarnessPrimeAgent AgentHarness = "prime-agent"
	HarnessAutohand   AgentHarness = "autohand"
	HarnessOMP        AgentHarness = "omp"
	// HarnessDeepSeekHarness is the DeepSeek Harness CLI ("dsh").
	HarnessDeepSeekHarness AgentHarness = "deepseek-harness"
	// HarnessFake is retained for existing test fixtures and historical session
	// rows, but is not user-selectable.
	HarnessFake AgentHarness = "fake"
)

// AllHarnesses lists every supported harness. It is the canonical set used to
// validate user-supplied harness names (e.g. per-project role overrides).
var AllHarnesses = []AgentHarness{
	HarnessClaudeCode, HarnessCodex, HarnessAider, HarnessOpenCode, HarnessGrok,
	HarnessDroid, HarnessAmp, HarnessAgy, HarnessCrush, HarnessCursor, HarnessQwen,
	HarnessCopilot, HarnessGoose, HarnessAuggie, HarnessContinue, HarnessDevin,
	HarnessCline, HarnessKimi, HarnessMuse, HarnessKiro, HarnessKilocode, HarnessVibe, HarnessPi,
	HarnessKimchi, HarnessPrimeAgent, HarnessAutohand,
	HarnessOMP,
	HarnessDeepSeekHarness,
}

// IsRecognizedPersisted reports whether h is an identity that existing durable
// rows and recovery paths may continue to read.
func (h AgentHarness) IsRecognizedPersisted() bool {
	if h == HarnessFake {
		return true
	}
	for _, k := range AllHarnesses {
		if h == k {
			return true
		}
	}
	return false
}

// IsSelectableForNewWork reports whether h may start new work in this build.
// Codex remains the recommended zero-configuration default; DeepSeek Harness
// is admitted alongside it as a worker once its profile-readiness checks pass,
// through the same fail-closed adapter admission (a missing dsh binary is
// "not ready", never silently skipped). opencode is admitted on the same terms:
// it resolves its own binary, reports authorization, and delivers the task
// prompt at launch, so a missing or unauthorized install fails closed here
// rather than producing a session that cannot run.
func (h AgentHarness) IsSelectableForNewWork() bool {
	return h == HarnessCodex || h == HarnessDeepSeekHarness || h == HarnessOpenCode
}

// IsSelectableAsCoordinator reports whether h may run as a project
// orchestrator — the Mission coordinator role. This is capability-gated, not a
// permanent provider allowlist: coordinating requires verified stable session
// identity, structured chat, and recovery support, which DeepSeek Harness has
// not demonstrated yet. When those capabilities pass their own admission, this
// predicate widens without touching the worker admission above.
//
// opencode satisfies all three: the workspace activity plugin reports its
// native session id (stable identity), it speaks ACP through a registered chat
// driver (structured chat), and it resumes that id with `--session` backed by a
// native-session probe (recovery).
func (h AgentHarness) IsSelectableAsCoordinator() bool {
	return h == HarnessCodex || h == HarnessOpenCode
}

// IsSelectableAsSwitchTarget reports whether h may be the destination of an
// agent switch on an existing logical session. Switching continues a prior
// conversation, so it requires verified continuation identity (native resume)
// and prompt-delivery support from the target adapter. DeepSeek Harness has
// neither yet — its restore path reports ok=false pending the dsh hook
// contract — so admitting it here would advertise switches that can only fail.
// Worker spawns are unaffected: they start fresh by design.
//
// opencode declares continuation capabilities (provider-assigned ids learned
// from its activity plugin), resolves the state root those ids live in, and
// probes that state before AO reuses a resume handle, so a switch onto it
// either continues a real conversation or truthfully starts fresh. It exposes
// no per-session transcript file — history lives in its SQLite state — so the
// switch records the source transcript as unavailable instead of inventing one.
func (h AgentHarness) IsSelectableAsSwitchTarget() bool {
	return h == HarnessCodex || h == HarnessOpenCode
}

// IsKnown is retained as a compatibility alias for persisted identity checks.
func (h AgentHarness) IsKnown() bool {
	return h.IsRecognizedPersisted()
}
