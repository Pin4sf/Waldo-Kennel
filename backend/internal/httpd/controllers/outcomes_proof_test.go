package controllers_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/config"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	outcomevc "github.com/aoagents/agent-orchestrator/backend/internal/service/outcome"
)

type fakeProofManager struct {
	get          func(context.Context, domain.OutcomeID) (outcomevc.ProofView, error)
	evidence     func(context.Context, domain.OutcomeID, outcomevc.RecordEvidenceInput) (outcomevc.ProofView, error)
	verification func(context.Context, domain.OutcomeID, outcomevc.RecordVerificationInput) (outcomevc.ProofView, error)
	decision     func(context.Context, domain.OutcomeID, outcomevc.DecideAcceptanceInput) (outcomevc.ProofView, error)
}

func (f *fakeProofManager) GetProof(ctx context.Context, id domain.OutcomeID) (outcomevc.ProofView, error) {
	return f.get(ctx, id)
}
func (f *fakeProofManager) RecordEvidence(ctx context.Context, id domain.OutcomeID, in outcomevc.RecordEvidenceInput) (outcomevc.ProofView, error) {
	return f.evidence(ctx, id, in)
}
func (f *fakeProofManager) RecordVerification(ctx context.Context, id domain.OutcomeID, in outcomevc.RecordVerificationInput) (outcomevc.ProofView, error) {
	return f.verification(ctx, id, in)
}
func (f *fakeProofManager) DecideAcceptance(ctx context.Context, id domain.OutcomeID, in outcomevc.DecideAcceptanceInput) (outcomevc.ProofView, error) {
	return f.decision(ctx, id, in)
}

func proofFixture() outcomevc.ProofView {
	revisionID := domain.ContractRevisionID("cr-proof")
	criterion := domain.ContractCriterion{ID: "crit-proof", ContractRevisionID: revisionID, Position: 1, Text: "It works."}
	return outcomevc.ProofView{
		OutcomeID: "out-proof",
		Contract: domain.ContractRevision{
			ID: revisionID, OutcomeID: "out-proof", Number: 1, Goal: "Prove it.",
			Criteria: []domain.ContractCriterion{criterion}, SuccessCriteria: []string{criterion.Text}, Review: "Owner review.",
		},
		Status: outcomevc.ProofStatusReadyForAcceptance, NextAction: "Review and accept.",
		Criteria: []outcomevc.CriterionProofView{{Criterion: criterion, Ready: true}},
	}
}

func newProofServer(t *testing.T, proof *fakeProofManager) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{Proof: proof}, httpd.ControlDeps{}))
}

func TestOutcomeProofRoutesUseTypedContract(t *testing.T) {
	fixture := proofFixture()
	manager := &fakeProofManager{}
	manager.get = func(_ context.Context, id domain.OutcomeID) (outcomevc.ProofView, error) {
		if id != fixture.OutcomeID {
			t.Fatalf("outcome id=%s", id)
		}
		return fixture, nil
	}
	manager.evidence = func(_ context.Context, id domain.OutcomeID, in outcomevc.RecordEvidenceInput) (outcomevc.ProofView, error) {
		if id != fixture.OutcomeID || in.ContractRevisionID != fixture.Contract.ID || in.CriterionID != "crit-proof" || in.SubjectType != domain.ProofSubjectOutcome || in.RequestKey != "ev-key" {
			t.Fatalf("evidence input=%+v id=%s", in, id)
		}
		return fixture, nil
	}
	manager.verification = func(_ context.Context, id domain.OutcomeID, in outcomevc.RecordVerificationInput) (outcomevc.ProofView, error) {
		if id != fixture.OutcomeID || len(in.EvidenceItemIDs) != 1 || in.IndependenceClass != domain.VerificationOwnerWalkthrough {
			t.Fatalf("verification input=%+v id=%s", in, id)
		}
		return fixture, nil
	}
	manager.decision = func(_ context.Context, id domain.OutcomeID, in outcomevc.DecideAcceptanceInput) (outcomevc.ProofView, error) {
		if id != fixture.OutcomeID || in.Kind != domain.AcceptanceAccept || in.ResourceDisposition != domain.ResourceDispositionRetain {
			t.Fatalf("decision input=%+v id=%s", in, id)
		}
		return fixture, nil
	}
	srv := newProofServer(t, manager)
	defer srv.Close()

	get, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/out-proof/proof", "")
	if status != http.StatusOK || !strings.Contains(string(get), `"criterionId":"crit-proof"`) || !strings.Contains(string(get), `"status":"ready_for_acceptance"`) {
		t.Fatalf("get proof status=%d body=%s", status, get)
	}

	evidenceBody := `{"expectedContractRevision":1,"contractRevisionId":"cr-proof","criterionId":"crit-proof","subjectType":"outcome","subjectId":"out-proof","subjectRevision":"cr-proof","kind":"supporting","sourceType":"owner_walkthrough","sourceRef":"walkthrough","producerType":"user","producerRef":"owner","summary":"Works.","contentDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","requestKey":"ev-key"}`
	if body, status, _ := doRequest(t, srv, http.MethodPost, "/api/v1/outcomes/out-proof/evidence", evidenceBody); status != http.StatusCreated {
		t.Fatalf("evidence status=%d body=%s", status, body)
	}
	verificationBody := `{"expectedContractRevision":1,"contractRevisionId":"cr-proof","criterionId":"crit-proof","subjectType":"outcome","subjectId":"out-proof","subjectRevision":"cr-proof","evidenceItemIds":["ev-proof"],"method":"Owner review.","independenceClass":"owner_walkthrough","result":"passed","verifierRef":"owner","requestKey":"ver-key"}`
	if body, status, _ := doRequest(t, srv, http.MethodPost, "/api/v1/outcomes/out-proof/verifications", verificationBody); status != http.StatusCreated {
		t.Fatalf("verification status=%d body=%s", status, body)
	}
	decisionBody := `{"expectedContractRevision":1,"contractRevisionId":"cr-proof","kind":"accept","summary":"Accepted.","resourceDisposition":"retain","requestKey":"acc-key"}`
	if body, status, _ := doRequest(t, srv, http.MethodPost, "/api/v1/outcomes/out-proof/acceptance-decisions", decisionBody); status != http.StatusCreated {
		t.Fatalf("decision status=%d body=%s", status, body)
	}
}

func TestOutcomeProofConflictPreservesRequestIDEnvelope(t *testing.T) {
	manager := &fakeProofManager{}
	manager.get = func(context.Context, domain.OutcomeID) (outcomevc.ProofView, error) {
		return outcomevc.ProofView{}, apierr.Conflict("OUTCOME_PROOF_CONTRACT_CONFLICT", "Contract moved", map[string]any{"currentRevision": 2})
	}
	srv := newProofServer(t, manager)
	defer srv.Close()

	body, status, _ := doRequest(t, srv, http.MethodGet, "/api/v1/outcomes/out-proof/proof", "")
	if status != http.StatusConflict {
		t.Fatalf("status=%d body=%s", status, body)
	}
	var response struct {
		Code      string `json:"code"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.Code != "OUTCOME_PROOF_CONTRACT_CONFLICT" || response.RequestID == "" {
		t.Fatalf("envelope=%+v err=%v body=%s", response, err, body)
	}
}

func TestOutcomeProofRoutesAnswer501WhenUnwired(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{}, httpd.ControlDeps{}))
	defer srv.Close()
	for _, route := range []string{"proof", "evidence", "verifications", "acceptance-decisions"} {
		method := http.MethodPost
		if route == "proof" {
			method = http.MethodGet
		}
		body, status, _ := doRequest(t, srv, method, "/api/v1/outcomes/out-proof/"+route, `{}`)
		if status != http.StatusNotImplemented || !strings.Contains(string(body), `"requestId"`) {
			t.Fatalf("%s status=%d body=%s", route, status, body)
		}
	}
}
