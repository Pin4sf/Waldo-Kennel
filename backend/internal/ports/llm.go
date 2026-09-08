package ports

import "context"

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
}

// LLMResponse carries the raw structured reply plus honest provenance. An
// adapter never fabricates EffectiveModel: an unknown value stays empty.
type LLMResponse struct {
	// JSON is the raw structured payload, already known to be valid JSON.
	JSON []byte
	// EffectiveModel is the model the provider reports actually serving the
	// request.
	EffectiveModel string
	// InputTokens and OutputTokens are provider-reported usage, zero when the
	// provider does not report it.
	InputTokens  int64
	OutputTokens int64
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
