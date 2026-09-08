package daemon

import (
	"os"
	"strings"
)

// waldoReasoningKey resolves the owner's own reasoning credential.
//
// The key belongs to the user and stays on this machine: ADR 0003 rules out a
// Waldo-operated LLM API and Waldo-funded inference, not Waldo reasoning with
// the owner's own authenticated provider. KENNEL_WALDO_API_KEY is preferred so
// an owner can keep Waldo's reasoning credential separate from the key their
// coding agents already use.
func waldoReasoningKey() string {
	for _, name := range []string{"KENNEL_WALDO_API_KEY", "ANTHROPIC_API_KEY"} {
		if key := strings.TrimSpace(os.Getenv(name)); key != "" {
			return key
		}
	}
	return ""
}
