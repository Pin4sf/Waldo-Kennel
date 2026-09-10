package codexappserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// IntelligenceProviderID is the provenance name for Codex-backed Waldo
// reasoning. It is intentionally not a domain.AgentHarness or an execution
// provider selection.
const IntelligenceProviderID = "codex-app-server"

const defaultIntelligenceTimeout = 2 * time.Minute

const intelligenceTurnAckTimeout = 5 * time.Second

// IntelligenceConfig selects the exact harness request. Empty Model and Effort
// preserve Codex's provider-default semantics; the adapter never fills either
// from an unrelated worker configuration.
type IntelligenceConfig struct {
	Model   string
	Effort  string
	Timeout time.Duration
}

// IntelligenceClient is a bounded, one-shot structured reasoning client over
// the existing Codex app-server transport. It creates an ephemeral provider
// thread per request and never creates a Kennel AgentSessionRef or WorkUnit.
type IntelligenceClient struct {
	driver *Driver
	cfg    IntelligenceConfig
}

var _ ports.LLMClient = (*IntelligenceClient)(nil)

// NewIntelligenceClient returns a Codex harness client. The caller owns the
// driver and must use the existing Codex plugin so authentication and binary
// resolution stay on one boundary.
func NewIntelligenceClient(driver *Driver, cfg IntelligenceConfig) *IntelligenceClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultIntelligenceTimeout
	}
	return &IntelligenceClient{driver: driver, cfg: cfg}
}

func (*IntelligenceClient) ID() string { return IntelligenceProviderID }

// Complete sends one native structured-output turn. Only the settled assistant
// message is returned to the intelligence service; transcript events never
// determine canonical proposal state.
func (c *IntelligenceClient) Complete(ctx context.Context, request ports.LLMRequest) (ports.LLMResponse, error) {
	if c == nil || c.driver == nil {
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningNotConfigured, "Codex harness reasoning is not configured", nil)
	}
	if c.driver.plugin == nil {
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningUnavailable, "The Codex harness plugin is unavailable", ports.ErrChatDriverUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return ports.LLMResponse{}, ports.ClassifyReasoningTransport(ctx, 0, err)
	}
	if request.SchemaName == "" || len(request.Schema) == 0 {
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningInvalidOutput, "Codex harness reasoning requires a structured output schema", nil)
	}

	callCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	// AuthStatus is deliberately advisory in the shared Chat driver. For this
	// one-shot reasoning path an affirmative unauthorized result is actionable,
	// while unknown continues to the app-server boundary where the real error is
	// observed.
	status, authErr := c.driver.plugin.AuthStatus(callCtx)
	if authErr == nil && status == ports.AgentAuthStatusUnauthorized {
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningUnauthorized, "Codex is not signed in; sign in with the Codex harness before asking Waldo to reason", ports.ErrChatAuthRequired)
	}

	workspace, err := os.MkdirTemp("", "kennel-codex-intelligence-")
	if err != nil {
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningUnavailable, "Waldo could not create disposable Codex reasoning state", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	conv, err := c.driver.startIntelligence(callCtx, workspace, request.System, c.cfg.Model)
	if err != nil {
		return ports.LLMResponse{}, classifyCodexReasoningFailure(callCtx, err)
	}

	schema, err := json.Marshal(request.Schema)
	if err != nil {
		_ = conv.Close()
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningInvalidOutput, "Waldo's structured output schema could not be encoded", err)
	}

	text := strings.TrimSpace(request.User)
	if text == "" {
		_ = conv.Close()
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningInvalidOutput, "Waldo's reasoning request is empty", nil)
	}
	text += "\n\nReturn only the JSON object required by the structured output schema. Do not use Markdown fences, commentary, tools, skills, MCP servers, or external effects."
	if request.MaxTokens > 0 {
		// Codex's app-server schema has no max-output-tokens field. Keep the
		// existing port's bounded intent visible to the model, while native
		// outputSchema remains the hard structural boundary.
		text += fmt.Sprintf(" Keep the JSON response below %d output tokens.", request.MaxTokens)
	}

	if err := ctx.Err(); err != nil {
		_ = conv.Close()
		return ports.LLMResponse{}, ports.ClassifyReasoningTransport(ctx, 0, err)
	}
	// Once turn/start has been issued, do not let caller cancellation erase the
	// provider turn id before the acknowledgement is read. The bounded wait
	// below will then interrupt that exact turn instead of leaving an ambiguous
	// provider operation alive.
	turnStartCtx := callCtx
	if deadline, ok := callCtx.Deadline(); ok {
		ackDeadline := time.Now().Add(intelligenceTurnAckTimeout)
		if deadline.Before(ackDeadline) {
			ackDeadline = deadline
		}
		var turnStartCancel context.CancelFunc
		turnStartCtx, turnStartCancel = context.WithDeadline(context.Background(), ackDeadline)
		defer turnStartCancel()
	}
	turn, err := conv.sendTurn(turnStartCtx, ports.ChatUserMessage{
		Text:            text,
		ClientMessageID: intelligenceMessageID(request, c.cfg),
		Origin:          domain.MessageOriginDaemon,
		Settings: ports.ChatTurnSettings{
			Model:  c.cfg.Model,
			Effort: c.cfg.Effort,
		},
	}, schema)
	if err != nil {
		_ = conv.Close()
		return ports.LLMResponse{}, classifyCodexReasoningFailure(callCtx, err)
	}

	response, err := waitForIntelligenceTurn(callCtx, conv, turn.ProviderTurnID)
	_ = conv.Close()
	if err != nil {
		return ports.LLMResponse{}, classifyCodexReasoningFailure(callCtx, err)
	}
	if !json.Valid(response) {
		return ports.LLMResponse{}, ports.NewReasoningFailure(
			ports.ReasoningInvalidOutput, "Codex returned a non-JSON structured reasoning reply", nil)
	}

	return ports.LLMResponse{
		JSON:             response,
		EffectiveModel:   conv.threadModel,
		NativeSessionRef: conv.ProviderConversationID(),
	}, nil
}

func (d *Driver) startIntelligence(ctx context.Context, workspace, system, model string) (*conversation, error) {
	if d == nil || d.plugin == nil {
		return nil, ports.ErrChatUnsupported
	}
	if !d.intelligenceBoundaryAvailable {
		return nil, fmt.Errorf("%w: Codex app-server has no proven no-tool or constrained-read boundary for Waldo reasoning", ports.ErrChatUnsupported)
	}
	if !strings.HasPrefix(workspace, string(os.PathSeparator)) {
		return nil, fmt.Errorf("intelligence workspace must be absolute")
	}
	conv, err := d.connect(ctx, workspace, nil)
	if err != nil {
		return nil, err
	}

	// These session flags reduce ambient context for the proposal-only thread,
	// but are not sufficient confinement proof on their own. The capability gate
	// above remains closed until tools and unrelated filesystem reads cannot
	// escape the approved packet.
	params := map[string]any{
		"cwd":            workspace,
		"approvalPolicy": "never",
		"sandbox":        "read-only",
		"ephemeral":      true,
		"config": map[string]any{
			"features":    map[string]any{"plugins": false, "apps": false},
			"skills":      map[string]any{"include_instructions": false},
			"mcp_servers": map[string]any{},
		},
	}
	if system = strings.TrimSpace(system); system != "" {
		params["developerInstructions"] = system
	}
	// Do not send an empty model: an empty model is the explicit provider-default
	// semantic, not permission to infer a model from another settings surface.
	if model = strings.TrimSpace(model); model != "" {
		params["model"] = model
	}
	var resp struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		Model           string `json:"model"`
		ReasoningEffort string `json:"reasoningEffort"`
	}
	if err := conv.conn.request(ctx, "thread/start", params, &resp); err != nil {
		_ = conv.Close()
		return nil, fmt.Errorf("thread/start: %w", err)
	}
	if resp.Thread.ID == "" {
		_ = conv.Close()
		return nil, errors.New("thread/start returned no thread id")
	}
	conv.start(resp.Thread.ID, resp.Model, resp.ReasoningEffort, nil)
	conv.intelligenceReadOnly = true
	return conv, nil
}

func waitForIntelligenceTurn(ctx context.Context, conv *conversation, turnID string) ([]byte, error) {
	if strings.TrimSpace(turnID) == "" {
		return nil, errors.New("Codex reasoning turn has no provider turn id")
	}
	var reply []byte
	for {
		select {
		case <-ctx.Done():
			interruptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			interruptErr := conv.Interrupt(interruptCtx, turnID)
			cancel()
			if interruptErr != nil {
				// The provider did not acknowledge interruption. The caller owns this
				// ephemeral app-server, so close it before returning; never represent
				// an unacknowledged interrupt as provider-side completion.
				_ = conv.Close()
			}
			return nil, ctx.Err()
		case ev, ok := <-conv.Events():
			if !ok {
				return nil, errors.New("Codex reasoning conversation ended before turn completion")
			}
			// A canceled context and an already-queued success event can both be
			// ready in this select. Recheck after receiving the event so cancellation
			// always wins over a response that arrived too late.
			if err := ctx.Err(); err != nil {
				interruptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				interruptErr := conv.Interrupt(interruptCtx, turnID)
				cancel()
				if interruptErr != nil {
					_ = conv.Close()
				}
				return nil, err
			}
			if ev.ProviderTurnID != turnID {
				// A stale or replayed event from another turn is not evidence for
				// this request. Identity-less events are rejected too: accepting one
				// would turn a parser omission into false evidence for this turn.
				continue
			}
			switch ev.Kind {
			case ports.ChatEventMessageCompleted:
				reply = []byte(strings.TrimSpace(ev.Text))
			case ports.ChatEventApprovalRequested, ports.ChatEventInputRequested:
				return nil, errors.New("Codex requested an unsupported interactive decision during bounded reasoning")
			case ports.ChatEventError:
				if ev.Err != nil {
					return nil, ev.Err
				}
				return nil, errors.New("Codex reasoning provider reported an error")
			case ports.ChatEventTurnCompleted:
				if ev.TurnState != domain.TurnStateCompleted {
					if ev.Err != nil {
						return nil, ev.Err
					}
					return nil, fmt.Errorf("Codex reasoning turn ended as %s", ev.TurnState)
				}
				if len(reply) == 0 {
					return nil, errors.New("Codex completed reasoning without a settled assistant message")
				}
				return reply, nil
			}
		}
	}
}

func intelligenceMessageID(request ports.LLMRequest, cfg IntelligenceConfig) string {
	h := sha256.New()
	for _, value := range []string{request.SchemaName, request.System, request.User, cfg.Model, cfg.Effort} {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(value))
	}
	return "waldo-" + hex.EncodeToString(h.Sum(nil))[:24]
}

func classifyCodexReasoningFailure(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil {
		if ctx.Err() != nil {
			return ports.ClassifyReasoningTransport(ctx, 0, err)
		}
	}
	switch {
	case errors.Is(err, ports.ErrChatAuthRequired):
		return ports.NewReasoningFailure(
			ports.ReasoningUnauthorized, "Codex is not signed in; sign in with the Codex harness before asking Waldo to reason", err)
	case errors.Is(err, ports.ErrChatUnsupported), errors.Is(err, ports.ErrChatDriverUnavailable), errors.Is(err, ports.ErrChatDriverIncompatible):
		return ports.NewReasoningFailure(
			ports.ReasoningUnavailable, "The installed Codex harness cannot provide bounded reasoning", err)
	default:
		return ports.NewReasoningFailure(
			ports.ReasoningUnavailable, "Codex harness reasoning failed before producing a usable proposal", err)
	}
}
