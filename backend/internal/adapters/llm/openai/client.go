// Package openai implements Waldo's reasoning surface against the OpenAI
// Responses API.
//
// Like the anthropic adapter beside it, this is deliberately NOT an
// agent/provider adapter. Agent adapters run the user's coding CLIs to execute
// authorized work; this one asks a model to propose structured material that
// the Go control plane then validates. Nothing here may create an Attempt,
// grant authority, or accept an Outcome.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	sdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// DefaultModel is the reasoning model Waldo uses unless the owner names
// another one. Waldo's own thinking is low-volume and correctness-sensitive,
// so it defaults to the most capable general model rather than the cheapest —
// the same judgement the anthropic adapter makes.
const DefaultModel = shared.ChatModelGPT6Astra

// defaultMaxTokens bounds a non-streaming reply. Reasoning models spend part
// of this budget on thinking that never reaches the output, so this sits above
// the anthropic adapter's ceiling for the same expected payload.
const defaultMaxTokens int64 = 32000

// ProviderID names this adapter in provenance records.
const ProviderID = "openai"

// Config is the owner-supplied reasoning credential and model choice. The key
// belongs to the user: Kennel never ships or funds inference (ADR 0003).
type Config struct {
	APIKey string
	Model  string
	// Effort tunes reasoning depth. Empty means the API default.
	Effort string
	// HTTPClient and BaseURL are injectable for controlled conformance tests.
	HTTPClient *http.Client
	BaseURL    string
	// MaxRetries is explicit so a paid/ambiguous reasoning call is never
	// duplicated by an SDK default. Production uses zero.
	MaxRetries int
}

// Client is a bounded, non-authoritative reasoning client.
type Client struct {
	api    sdk.Client
	model  string
	effort shared.ReasoningEffort
}

var _ ports.LLMClient = (*Client)(nil)

// ErrNotConfigured reports that no reasoning credential is available. Waldo
// cannot think without one, and the caller must surface that truthfully rather
// than silently degrading to a canned answer.
//
// It is a classified ReasoningFailure so the setup state survives the whole way
// to the API as an actionable code instead of an opaque 500. Identity still
// works with errors.Is because this is a single package-level value.
var ErrNotConfigured error = ports.NewReasoningFailure(
	ports.ReasoningNotConfigured, "No Waldo reasoning key is configured", nil)

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
	// The SDK would otherwise fall back to OPENAI_API_KEY from the ambient
	// environment. Waldo resolves its own credential explicitly, so pass it in
	// and keep which key was used a decision the daemon made, not the SDK.
	options := []option.RequestOption{option.WithAPIKey(key), option.WithMaxRetries(cfg.MaxRetries)}
	if cfg.HTTPClient != nil {
		options = append(options, option.WithHTTPClient(cfg.HTTPClient))
	}
	if strings.TrimSpace(cfg.BaseURL) != "" {
		options = append(options, option.WithBaseURL(strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")))
	}
	return &Client{
		api:    sdk.NewClient(options...),
		model:  model,
		effort: shared.ReasoningEffort(strings.TrimSpace(cfg.Effort)),
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

	params := responses.ResponseNewParams{
		Model:           c.model,
		MaxOutputTokens: sdk.Int(maxTokens),
		Input:           responses.ResponseNewParamsInputUnion{OfString: sdk.String(req.User)},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:   schemaName(req.SchemaName),
					Schema: strictSchema(req.Schema),
					Strict: sdk.Bool(true),
				},
			},
		},
	}
	if system := strings.TrimSpace(req.System); system != "" {
		params.Instructions = sdk.String(system)
		// The system block is stable per call kind, so bucketing by call kind
		// keeps repeated Contract/Plan reasoning on a warm prefix cache.
		params.PromptCacheKey = sdk.String("kennel-waldo-" + schemaName(req.SchemaName))
	}
	if c.effort != "" {
		params.Reasoning = shared.ReasoningParam{Effort: c.effort}
	}

	response, err := c.api.Responses.New(ctx, params)
	if err != nil {
		// The SDK carries the HTTP status on its own error type; the shared
		// classifier owns the status-to-meaning rule so both adapters agree.
		return ports.LLMResponse{}, ports.ClassifyReasoningTransport(ctx, statusOf(err), err)
	}
	if refusal := refusalOf(response); refusal != "" {
		return ports.LLMResponse{}, ports.NewReasoningFailure(ports.ReasoningDeclined,
			fmt.Sprintf("Waldo reasoning was declined (%s)", refusal), nil)
	}
	switch response.Status {
	case responses.ResponseStatusFailed:
		return ports.LLMResponse{}, ports.NewReasoningFailure(ports.ReasoningUnavailable,
			fmt.Sprintf("Waldo reasoning failed: %s", response.Error.Message), nil)
	case responses.ResponseStatusIncomplete:
		// Most often the token budget ran out mid-object. Say which, because
		// the owner's fix differs: raise the budget, or shorten the input.
		return ports.LLMResponse{}, ports.NewReasoningFailure(ports.ReasoningIncomplete,
			fmt.Sprintf("Waldo reasoning stopped before a complete %s payload (%s)",
				req.SchemaName, response.IncompleteDetails.Reason), nil)
	}

	raw := strings.TrimSpace(response.OutputText())
	if raw == "" {
		return ports.LLMResponse{}, ports.NewReasoningFailure(ports.ReasoningInvalidOutput,
			fmt.Sprintf("Waldo reasoning returned no %s payload", req.SchemaName), nil)
	}
	if !json.Valid([]byte(raw)) {
		return ports.LLMResponse{}, ports.NewReasoningFailure(ports.ReasoningInvalidOutput,
			fmt.Sprintf("Waldo reasoning returned malformed %s JSON", req.SchemaName), nil)
	}

	return ports.LLMResponse{
		JSON:           []byte(raw),
		EffectiveModel: response.Model,
		InputTokens:    int64Ptr(response.Usage.InputTokens),
		OutputTokens:   int64Ptr(response.Usage.OutputTokens),
	}, nil
}

func int64Ptr(value int64) *int64 { return &value }

// refusalOf reports a model refusal, which the Responses API returns as a
// content part rather than an error.
func refusalOf(response *responses.Response) string {
	if response == nil {
		return ""
	}
	for _, item := range response.Output {
		for _, content := range item.Content {
			if content.Type == "refusal" && strings.TrimSpace(content.Refusal) != "" {
				return content.Refusal
			}
		}
	}
	return ""
}

// schemaName coerces a caller's schema name into the identifier shape the API
// accepts: letters, digits, underscores and dashes, at most 64 characters.
func schemaName(name string) string {
	var out strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			out.WriteRune(r)
		default:
			out.WriteRune('_')
		}
		if out.Len() >= 64 {
			break
		}
	}
	if out.Len() == 0 {
		return "reply"
	}
	return out.String()
}

// statusOf extracts the HTTP status the SDK recorded on a failed call, or zero
// when the failure never reached a response (dial, deadline, cancellation).
func statusOf(err error) int {
	var apiErr *sdk.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}
