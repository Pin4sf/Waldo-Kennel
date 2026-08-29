package outcome

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// DecompositionRequestView is one ask and what became of it.
type DecompositionRequestView struct {
	Request domain.DecompositionRequest
	// Expired is derived from the clock rather than stored, so a request that
	// timed out while the daemon was down still reads as expired.
	Expired bool
}

// AskForDecomposition opens a durable request and starts an agent working on
// it. It returns immediately: the proposal arrives later, over the API.
func (s *Service) AskForDecomposition(ctx context.Context, outcomeID domain.OutcomeID, expectedContractRevision int64) (DecompositionRequestView, error) {
	if s.proposer == nil {
		return DecompositionRequestView{}, apierr.Internal("DECOMPOSITION_PROPOSER_UNWIRED",
			"Agent-authored decomposition is not wired in this environment")
	}
	parent, current, err := s.decomposableParent(ctx, outcomeID, expectedContractRevision)
	if err != nil {
		return DecompositionRequestView{}, err
	}
	now := s.clock()

	// One open ask at a time. Two agents answering the same Outcome would
	// race to produce competing proposals, and the second would be refused as
	// a closed request anyway — better to say so before spawning it.
	if existing, found, err := s.store.LatestDecompositionRequest(ctx, outcomeID); err != nil {
		return DecompositionRequestView{}, err
	} else if found && existing.Status.Open() && !existing.Expired(now) {
		return DecompositionRequestView{}, apierr.Conflict("DECOMPOSITION_REQUEST_OPEN",
			"An agent is already working on a decomposition for this Outcome",
			map[string]any{"requestId": string(existing.ID), "expiresAt": existing.ExpiresAt})
	}

	token, err := newCallbackToken()
	if err != nil {
		return DecompositionRequestView{}, apierr.Internal("DECOMPOSITION_TOKEN_FAILED", "Could not mint a callback token")
	}
	request := domain.DecompositionRequest{
		ID:                  domain.DecompositionRequestID("dreq-" + uuid.NewString()),
		OutcomeID:           parent.ID,
		ContractRevisionID:  current.ID,
		Status:              domain.DecompositionRequested,
		CallbackTokenDigest: domain.HashCallbackToken(token),
		ExpiresAt:           now.Add(domain.DefaultDecompositionRequestTTL),
		CreatedAt:           now,
	}

	projectID, ok, err := s.store.GetOutcomeProjectID(ctx, outcomeID)
	if err != nil {
		return DecompositionRequestView{}, err
	}
	if !ok {
		return DecompositionRequestView{}, apierr.NotFound("PROJECT_NOT_FOUND", "Register that project before asking for a decomposition")
	}

	// Persist BEFORE spawning. A spawned agent holding a token for a request
	// that was never recorded would have nowhere to answer, and its work would
	// be silently lost.
	if err := s.store.CreateDecompositionRequest(ctx, request); err != nil {
		return DecompositionRequestView{}, err
	}

	ticket, err := s.proposer.Propose(ctx, ports.DecompositionProposalInput{
		RequestID:        request.ID,
		ProjectID:        projectID,
		OutcomeID:        parent.ID,
		OutcomeTitle:     parent.Title,
		Contract:         current,
		CallbackToken:    token,
		MaxContributions: domain.MaxProposedContributions,
		ParentAuthority:  current.AuthorityCeiling,
	})
	if err != nil {
		// The spawn failed, so nothing will ever answer. Close the request now
		// rather than leaving the owner watching a request with no agent.
		_ = s.store.AnswerDecompositionRequest(ctx, ports.DecompositionRequestAnswer{
			RequestID: request.ID, Status: domain.DecompositionRejected,
			RefusalReason: "Could not start an agent to propose: " + err.Error(), At: s.clock(),
		})
		return DecompositionRequestView{}, apierr.Internal("DECOMPOSITION_SPAWN_FAILED", err.Error())
	}
	request.SessionID = ticket.SessionID
	return DecompositionRequestView{Request: request}, nil
}

// SubmitAgentProposal is the callback an agent-authored proposal arrives on.
//
// Routing is checked BEFORE the proposal is parsed: an answer to a closed,
// expired, or wrongly-addressed request is a routing problem, not a validation
// one. Past that, the draft goes through exactly the same gates a
// hand-authored proposal does — there is no trusted-proposer path.
func (s *Service) SubmitAgentProposal(
	ctx context.Context,
	requestID domain.DecompositionRequestID,
	token string,
	in ProposeDecompositionInput,
	raw string,
) (DecompositionRequestView, error) {
	request, found, err := s.store.GetDecompositionRequest(ctx, requestID)
	if err != nil {
		return DecompositionRequestView{}, err
	}
	if !found {
		return DecompositionRequestView{}, apierr.NotFound("DECOMPOSITION_REQUEST_NOT_FOUND", "That decomposition request does not exist")
	}
	now := s.clock()
	if err := request.AdmitProposalAnswer(token, now); err != nil {
		// Deliberately the same shape for every routing refusal: a caller
		// probing tokens learns only that its answer was not admitted.
		return DecompositionRequestView{}, apierr.Conflict("DECOMPOSITION_REQUEST_NOT_ADMITTED", err.Error(),
			map[string]any{"requestId": string(requestID)})
	}
	if len(in.Contributors) > domain.MaxProposedContributions {
		return s.rejectProposal(ctx, request, raw, fmt.Sprintf(
			"A proposal may contain at most %d contributing Outcomes; this one had %d",
			domain.MaxProposedContributions, len(in.Contributors)))
	}

	// The request froze a contract revision. A proposal answering a superseded
	// one is refused rather than rebound to whatever is current now.
	in.ExpectedContractRevision = 0
	parent, found, err := s.store.GetOutcome(ctx, request.OutcomeID)
	if err != nil {
		return DecompositionRequestView{}, err
	}
	if !found {
		return DecompositionRequestView{}, apierr.NotFound("OUTCOME_NOT_FOUND", "That Outcome no longer exists")
	}
	current, err := s.currentRevision(ctx, parent)
	if err != nil {
		return DecompositionRequestView{}, err
	}
	if current.ID != request.ContractRevisionID {
		return s.rejectProposal(ctx, request, raw,
			"The contract changed while the agent was working; this proposal answers a superseded revision")
	}
	in.ExpectedContractRevision = current.Number

	proposed, err := s.ProposeDecomposition(ctx, request.OutcomeID, in)
	if err != nil {
		// The daemon's own words are kept with the draft so the owner can see
		// exactly what was wrong and fix that one thing in the editor.
		return s.rejectProposal(ctx, request, raw, apierrMessage(err))
	}

	if err := s.store.AnswerDecompositionRequest(ctx, ports.DecompositionRequestAnswer{
		RequestID: request.ID, Status: domain.DecompositionFulfilled,
		RawProposal: raw, DecompositionID: proposed.Decomposition.ID, At: now,
	}); err != nil {
		if errors.Is(err, ports.ErrDecompositionRequestClosed) {
			return DecompositionRequestView{}, apierr.Conflict("DECOMPOSITION_REQUEST_NOT_ADMITTED",
				"This decomposition request was answered concurrently", nil)
		}
		return DecompositionRequestView{}, err
	}
	request.Status = domain.DecompositionFulfilled
	request.DecompositionID = proposed.Decomposition.ID
	request.RawProposal = raw
	request.AnsweredAt = &now
	return DecompositionRequestView{Request: request}, nil
}

// rejectProposal records a refusal WITH the draft, so the owner corrects one
// field instead of regenerating. The draft is stored as opaque JSON on the
// request and never as a DecompositionRevision.
func (s *Service) rejectProposal(ctx context.Context, request domain.DecompositionRequest, raw, reason string) (DecompositionRequestView, error) {
	now := s.clock()
	if err := s.store.AnswerDecompositionRequest(ctx, ports.DecompositionRequestAnswer{
		RequestID: request.ID, Status: domain.DecompositionRejected,
		RawProposal: raw, RefusalReason: reason, At: now,
	}); err != nil && !errors.Is(err, ports.ErrDecompositionRequestClosed) {
		return DecompositionRequestView{}, err
	}
	request.Status = domain.DecompositionRejected
	request.RawProposal = raw
	request.RefusalReason = reason
	request.AnsweredAt = &now
	return DecompositionRequestView{Request: request}, nil
}

// LatestDecompositionRequest reads an Outcome's newest ask.
func (s *Service) LatestDecompositionRequest(ctx context.Context, outcomeID domain.OutcomeID) (DecompositionRequestView, error) {
	request, found, err := s.store.LatestDecompositionRequest(ctx, outcomeID)
	if err != nil {
		return DecompositionRequestView{}, err
	}
	if !found {
		return DecompositionRequestView{}, apierr.NotFound("DECOMPOSITION_REQUEST_NOT_FOUND", "This Outcome has not been asked")
	}
	return DecompositionRequestView{Request: request, Expired: request.Expired(s.clock())}, nil
}

// ExpireStaleDecompositionRequests closes asks whose deadline passed. It runs
// at startup and on a timer, because expiry is a durable deadline rather than
// an in-memory timer: a request that timed out while the daemon was down still
// has to reach a verdict.
func (s *Service) ExpireStaleDecompositionRequests(ctx context.Context) (int, error) {
	open, err := s.store.ListOpenDecompositionRequests(ctx)
	if err != nil {
		return 0, err
	}
	now := s.clock()
	closed := 0
	for _, request := range open {
		if !request.Expired(now) {
			continue
		}
		if err := s.store.AnswerDecompositionRequest(ctx, ports.DecompositionRequestAnswer{
			RequestID: request.ID, Status: domain.DecompositionExpired,
			RefusalReason: "No proposal arrived before the request expired", At: now,
		}); err != nil && !errors.Is(err, ports.ErrDecompositionRequestClosed) {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

// newCallbackToken mints the scoping token handed to a spawned session.
//
// It is NOT authentication: the loopback listener is unauthenticated by
// deliberate decision, so any local process could already reach the callback.
// It scopes an answer to one request, single-use and expiring, which is what
// stops a confused or retrying agent answering for the wrong Outcome.
func newCallbackToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// apierrMessage extracts the daemon's own refusal text.
func apierrMessage(err error) string {
	var apiErr *apierr.Error
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	return err.Error()
}

// MarshalRawProposal renders a submitted draft for retention. A draft that
// cannot be rendered is stored as the empty string rather than failing the
// refusal it belongs to.
func MarshalRawProposal(in ProposeDecompositionInput) string {
	body, err := json.Marshal(in)
	if err != nil {
		return ""
	}
	return string(body)
}
