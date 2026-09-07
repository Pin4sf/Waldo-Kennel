package usage

import "github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"

// SupportedHarness reports whether the harness has a certified usage pipeline.
func SupportedHarness(h domain.AgentHarness) bool {
	switch h {
	case domain.HarnessClaudeCode, domain.HarnessCodex:
		return true
	default:
		return false
	}
}
