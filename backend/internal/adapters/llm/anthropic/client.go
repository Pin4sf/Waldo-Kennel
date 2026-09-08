// Package anthropic implements Waldo's reasoning surface against the Claude
// Messages API.
//
// This adapter is deliberately NOT an agent/provider adapter. Agent adapters
// run the user's coding CLIs to execute authorized work; this one asks a model
// to propose structured material that the Go control plane then validates.
// Nothing here may create an Attempt, grant authority, or accept an Outcome.
package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// DefaultModel is the reasoning model Waldo uses unless the owner names
// another one. Waldo's own thinking is low-volume and correctness-sensitive,
// so it defaults to the most capable general model rather than the cheapest.
const DefaultModel = "claude-opus-5"

// defaultMaxTokens keeps a non-streaming reply comfortably inside the SDK's
// HTTP timeout.
const defaultMaxTokens int64 = 16000

// ProviderID names this adapter in provenance records.
const ProviderID = "anthropic"

// Config is the owner-supplied reasoning credential and model choice. The key
// belongs to the user: Kennel never ships or funds inference (ADR 0003).
type Config struct {
	APIKey string
	Model  string
	// Effort tunes reasoning depth. Empty means the API default.
	Effort string
}

// Client is a bounded, non-authoritative reasoning client.
type Client struct {
	api    sdk.Client
	model  string
	effort sdk.OutputConfigEffort
}

var _ ports.LLMClient = (*Client)(nil)

// ErrNotConfigured reports that no reasoning credential is available. Waldo
// cannot think without one, and the caller must surface that truthfully rather
// than silently degrading to a canned answer.
var ErrNotConfigured = errors.New("no Waldo reasoning key is configured")

// New builds a client, or returns ErrNotConfigured when no key is present.
func New(cfg Config) (*Client, error) {
	key := strings.TrimSpace(cfg.APIKey)
	if key == "" {
		return nil, ErrNotConfigured
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		api:    sdk.NewClient(option.WithAPIKey(key)),
		model:  model,
		effort: sdk.OutputConfigEffort(strings.TrimSpace(cfg.Effort)),
	}, nil
}

// ID names the provider for provenance.
func (*Client) ID() string { return ProviderID }

// Complete performs one structured reasoning call. The reply is constrained by
// the caller's JSON Schema on the provider side and re-validated as JSON here;
// domain validation stays with the Go control plane.
func (c *Client) Complete(ctx context.Context, req ports.LLMRequest) (ports.LLMResponse, error) {
	if c == nil {
		return ports.LLMResponse{}, ErrNotConfigured
	}
	if len(req.Schema) == 0 {
		return ports.LLMResponse{}, fmt.Errorf("reasoning request %q requires an output schema", req.SchemaName)
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	params := sdk.MessageNewParams{
		Model:     sdk.Model(c.model),
		MaxTokens: maxTokens,
		Messages: []sdk.MessageParam{
			sdk.NewUserMessage(sdk.NewTextBlock(req.User)),
		},
		OutputConfig: sdk.OutputConfigParam{
			Format: sdk.JSONOutputFormatParam{Schema: req.Schema},
		},
	}
	if system := strings.TrimSpace(req.System); system != "" {
		// The system block is stable per call kind, so caching it keeps
		// repeated Contract/Plan reasoning cheap.
		params.System = []sdk.TextBlockParam{{
			Text:         system,
			CacheControl: sdk.NewCacheControlEphemeralParam(),
		}}
	}
	if c.effort != "" {
		params.OutputConfig.Effort = c.effort
	}

	message, err := c.api.Messages.New(ctx, params)
	if err != nil {
		return ports.LLMResponse{}, fmt.Errorf("waldo reasoning call failed: %w", err)
	}
	if message.StopReason == sdk.StopReasonRefusal {
		return ports.LLMResponse{}, fmt.Errorf("waldo reasoning was declined (%s)", message.StopDetails.Category)
	}

	var body strings.Builder
	for _, block := range message.Content {
		if text, ok := block.AsAny().(sdk.TextBlock); ok {
			body.WriteString(text.Text)
		}
	}
	raw := strings.TrimSpace(body.String())
	if raw == "" {
		return ports.LLMResponse{}, fmt.Errorf("waldo reasoning returned no %s payload", req.SchemaName)
	}
	if !json.Valid([]byte(raw)) {
		return ports.LLMResponse{}, fmt.Errorf("waldo reasoning returned malformed %s JSON", req.SchemaName)
	}

	return ports.LLMResponse{
		JSON:           []byte(raw),
		EffectiveModel: string(message.Model),
		InputTokens:    message.Usage.InputTokens,
		OutputTokens:   message.Usage.OutputTokens,
	}, nil
}
