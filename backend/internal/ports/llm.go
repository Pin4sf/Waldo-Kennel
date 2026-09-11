package ports

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// ReasoningContextMode identifies the filesystem context a native reasoning
// adapter may expose to model tools. The zero value is packet-only: request
// text may contain a bounded snapshot, but tools receive no Project access.
type ReasoningContextMode string

const (
	// ReasoningContextPacketOnly exposes only the bounded context rendered into
	// the request text; native tools receive no Project root.
	ReasoningContextPacketOnly ReasoningContextMode = ""
	// ReasoningContextRepositoryRead exposes the selected Project root through
	// the native provider's read-only investigation boundary.
	ReasoningContextRepositoryRead ReasoningContextMode = "repository_read"
)

// ReasoningContextAccess is request-scoped authority for native reasoning.
// It never grants writes, arbitrary network access, or execution authority.
type ReasoningContextAccess struct {
	Mode ReasoningContextMode
	Root string
}

// Validate rejects ambiguous native context rather than letting an adapter
// infer authority from prompt text or the presence of a repository packet.
func (a ReasoningContextAccess) Validate() error {
	switch a.Mode {
	case ReasoningContextPacketOnly:
		if strings.TrimSpace(a.Root) != "" {
			return fmt.Errorf("packet-only reasoning cannot name a repository root")
		}
		return nil
	case ReasoningContextRepositoryRead:
		if !filepath.IsAbs(strings.TrimSpace(a.Root)) {
			return fmt.Errorf("repository-read reasoning requires an absolute root")
		}
		return nil
	default:
		return fmt.Errorf("unsupported reasoning context mode %q", a.Mode)
	}
}

// LLMRequest is one bounded, non-authoritative reasoning call. Kennel never
// sends authority, acceptance, or fencing decisions through this port: the
// model proposes structured material and the Go control plane validates it.
type LLMRequest struct {
	// System is the operator instruction. It is stable across calls of the
	// same kind so provider prompt caches stay warm.
	System string
	// User is the request-specific input.
	User string
	// SchemaName names the JSON shape the model must return.
	SchemaName string
	// Schema is the JSON Schema the reply is validated against by the
	// provider where supported, and by Kennel unconditionally.
	Schema map[string]any
	// MaxTokens bounds the reply. Zero means the adapter's default.
	MaxTokens int64
	// ContextAccess carries explicit, provider-neutral native tool authority.
	// Direct API adapters use the already-rendered packet and ignore this field.
	ContextAccess ReasoningContextAccess
}

// LLMResponse carries the raw structured reply plus honest provenance. An
// adapter never fabricates EffectiveModel: an unknown value stays empty.
type LLMResponse struct {
	// JSON is the raw structured payload, already known to be valid JSON.
	JSON []byte
	// EffectiveModel is the model the provider reports actually serving the
	// request.
	EffectiveModel string
	// NativeSessionRef identifies a provider-native proposal session when the
	// adapter used one. It is provenance only; it is not an AgentSessionRef and
	// grants no execution authority.
	NativeSessionRef string
	// InputTokens and OutputTokens are provider-reported usage, zero when the
	// provider does not report it.
	// Pointers distinguish provider-reported zero from usage that was not
	// reported. Unknown usage must never be rendered as free/zero usage.
	InputTokens  *int64
	OutputTokens *int64
}

// LLMClient is Waldo's own reasoning surface. It is deliberately separate from
// the provider/agent adapters that execute authorized work: coding agents
// execute, Waldo thinks, and neither borrows the other's authority.
type LLMClient interface {
	// ID names the configured provider, for provenance only.
	ID() string
	// Complete performs one structured reasoning call.
	Complete(context.Context, LLMRequest) (LLMResponse, error)
}
