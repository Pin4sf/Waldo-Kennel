package store_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
)

func TestOutcomeProofStorePersistsCriteriaAndProofAcrossRestart(t *testing.T) {
	dataDir := t.TempDir()
	s, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "proof-project")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "proof-project")
	if err != nil {
		t.Fatalf("ensure work space: %v", err)
	}
	now := time.Date(2026, 8, 25, 19, 0, 0, 0, time.UTC)
	revisionID := domain.ContractRevisionID("cr-proof-1")
	outcome := domain.Outcome{ID: "out-proof", SpaceID: space.ID, Title: "Local Focus Ledger"}
	revision := domain.ContractRevision{
		ID:              revisionID,
		OutcomeID:       outcome.ID,
		Goal:            "Record and retain focus time.",
		SuccessCriteria: []string{"Record one block.", "Survive restart."},
		Criteria: []domain.ContractCriterion{
			{ID: "crit-record", ContractRevisionID: revisionID, Position: 1, Text: "Record one block."},
			{ID: "crit-restart", ContractRevisionID: revisionID, Position: 2, Text: "Survive restart."},
		},
		Review:    "Checks and owner walkthrough.",
		CreatedAt: now,
	}
	if err := s.CreateOutcomeWithContract(ctx, outcome, revision, "outcome-proof-key"); err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	history, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil || len(history) != 1 || len(history[0].Criteria) != 2 {
		t.Fatalf("criterion read-back history=%+v err=%v", history, err)
	}
	if history[0].Criteria[1].ID != "crit-restart" || history[0].Criteria[1].ContractRevisionID != revisionID {
		t.Fatalf("stable criterion identity lost: %+v", history[0].Criteria[1])
	}

	evidence := domain.EvidenceItem{
		ID: "ev-proof", OutcomeID: outcome.ID, ContractRevisionID: revisionID, CriterionID: "crit-restart",
		SubjectType: domain.ProofSubjectOutcome, SubjectID: string(outcome.ID), SubjectRevision: string(revisionID),
		Kind: domain.EvidenceSupporting, SourceType: domain.EvidenceSourceOwnerWalkthrough,
		SourceRef: "walkthrough-1", ProducerType: domain.EvidenceProducerUser, ProducerRef: "owner",
		Summary: "The entry remained after restart.", ContentDigest: strings.Repeat("a", 64),
		RequestKey: "evidence-proof-key", RequestFingerprint: strings.Repeat("b", 64), CreatedAt: now.Add(time.Minute),
	}
	if err := s.CreateEvidenceItem(ctx, evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
	verification := domain.VerificationRun{
		ID: "ver-proof", OutcomeID: outcome.ID, ContractRevisionID: revisionID, CriterionID: "crit-restart",
		SubjectType: evidence.SubjectType, SubjectID: evidence.SubjectID, SubjectRevision: evidence.SubjectRevision,
		EvidenceItemIDs: []domain.EvidenceItemID{evidence.ID}, Method: "Owner restart walkthrough.",
		IndependenceClass: domain.VerificationOwnerWalkthrough, Result: domain.VerificationPassed,
		VerifierRef: "owner", RequestKey: "verification-proof-key",
		RequestFingerprint: strings.Repeat("c", 64), CreatedAt: now.Add(2 * time.Minute),
	}
	if err := s.CreateVerificationRun(ctx, verification); err != nil {
		t.Fatalf("create verification: %v", err)
	}
	decision := domain.AcceptanceDecision{
		ID: "acc-proof", OutcomeID: outcome.ID, ContractRevisionID: revisionID,
		Kind: domain.AcceptanceRequestRework, ActorType: domain.AcceptanceActorUser,
		Summary: "The date-boundary proof is still missing.", ResourceDisposition: domain.ResourceDispositionRetain,
		RequestKey: "decision-proof-key", RequestFingerprint: strings.Repeat("d", 64), CreatedAt: now.Add(3 * time.Minute),
	}
	correction := domain.OutcomeCorrection{
		ID: "corr-proof", DecisionID: decision.ID, OutcomeID: outcome.ID, ContractRevisionID: revisionID,
		Feedback: decision.Summary, TargetType: domain.ReentryTargetContract, TargetID: string(revisionID),
		CreatedAt: decision.CreatedAt,
	}
	if err := s.CreateAcceptanceDecision(ctx, decision, &correction); err != nil {
		t.Fatalf("create decision + correction: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}
	s, err = sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	evidenceRows, err := s.ListEvidenceItems(ctx, outcome.ID)
	if err != nil || len(evidenceRows) != 1 || evidenceRows[0].CriterionID != "crit-restart" {
		t.Fatalf("evidence after restart=%+v err=%v", evidenceRows, err)
	}
	verificationRows, err := s.ListVerificationRuns(ctx, outcome.ID)
	if err != nil || len(verificationRows) != 1 || verificationRows[0].EvidenceItemIDs[0] != evidence.ID {
		t.Fatalf("verification after restart=%+v err=%v", verificationRows, err)
	}
	decisions, err := s.ListAcceptanceDecisions(ctx, outcome.ID)
	if err != nil || len(decisions) != 1 || decisions[0].ActorType != domain.AcceptanceActorUser {
		t.Fatalf("decisions after restart=%+v err=%v", decisions, err)
	}
	corrections, err := s.ListOutcomeCorrections(ctx, outcome.ID)
	if err != nil || len(corrections) != 1 || corrections[0].TargetID != string(revisionID) {
		t.Fatalf("corrections after restart=%+v err=%v", corrections, err)
	}
}

func TestOutcomeProofStoreIdempotencyLookupPreservesOriginalFingerprint(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	seedProject(t, s, "proof-idempotency")
	space, _ := s.EnsureWorkResponsibilitySpace(ctx, "proof-idempotency")
	revisionID := domain.ContractRevisionID("cr-idem")
	outcome := domain.Outcome{ID: "out-idem", SpaceID: space.ID, Title: "Idempotent proof"}
	revision := domain.ContractRevision{
		ID: revisionID, OutcomeID: outcome.ID, Goal: "Prove once.",
		SuccessCriteria: []string{"One proof row."},
		Criteria:        []domain.ContractCriterion{{ID: "crit-idem", ContractRevisionID: revisionID, Position: 1, Text: "One proof row."}},
		Review:          "Read it back.", CreatedAt: time.Now().UTC(),
	}
	if err := s.CreateOutcomeWithContract(ctx, outcome, revision, "out-idem-key"); err != nil {
		t.Fatalf("create outcome: %v", err)
	}
	evidence := domain.EvidenceItem{
		ID: "ev-idem", OutcomeID: outcome.ID, ContractRevisionID: revisionID, CriterionID: "crit-idem",
		SubjectType: domain.ProofSubjectOutcome, SubjectID: string(outcome.ID), SubjectRevision: string(revisionID),
		Kind: domain.EvidenceSupporting, SourceType: domain.EvidenceSourceArtifact, SourceRef: "artifact-1",
		ProducerType: domain.EvidenceProducerUser, ProducerRef: "owner", Summary: "Proof.",
		ContentDigest: strings.Repeat("e", 64), RequestKey: "same-key", RequestFingerprint: strings.Repeat("f", 64),
	}
	if err := s.CreateEvidenceItem(ctx, evidence); err != nil {
		t.Fatalf("create evidence: %v", err)
	}
	got, ok, err := s.FindEvidenceItemByRequestKey(ctx, "same-key")
	if err != nil || !ok || got.ID != evidence.ID || got.RequestFingerprint != evidence.RequestFingerprint {
		t.Fatalf("lookup got=%+v ok=%v err=%v", got, ok, err)
	}
	conflict := evidence
	conflict.ID = "ev-conflict"
	conflict.RequestFingerprint = strings.Repeat("0", 64)
	if err := s.CreateEvidenceItem(ctx, conflict); err == nil {
		t.Fatal("duplicate request key must be rejected so service can resolve replay or conflict")
	}
	rows, err := s.ListEvidenceItems(ctx, outcome.ID)
	if err != nil || len(rows) != 1 || rows[0].ID != evidence.ID {
		t.Fatalf("rows after conflicting replay=%+v err=%v", rows, err)
	}
}
