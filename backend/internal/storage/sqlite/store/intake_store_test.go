package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func TestIntakeStorePersistsAppendOnlyProposalAndConversationRefsAcrossRestart(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, store, "intake-project")
	now := time.Date(2026, 8, 26, 5, 0, 0, 0, time.UTC)
	session := domain.IntakeSession{
		ID: "intake-restart", SourceSurface: domain.IntakeSourceHome, Purpose: domain.IntakePurposeOutcome,
		ProjectID: "intake-project", SourceOpenLoopID: "loop-1", Statement: "Make restart durable",
		Status: domain.IntakeStatusCaptured, CreatedAt: now, UpdatedAt: now,
	}
	created, err := store.CreateIntake(ctx, session, []domain.IntakeConversationRef{{EpisodeID: "episode-1", TurnID: "turn-2", Position: 1}}, ports.IntakeIdempotency{Key: "capture-restart", Fingerprint: "fp-capture"})
	if err != nil {
		t.Fatalf("CreateIntake() error = %v", err)
	}
	if _, err := store.BeginIntakeAnalysis(ctx, created.Session.ID, 0, now); err != nil {
		t.Fatalf("BeginIntakeAnalysis() error = %v", err)
	}
	proposal := storeProposal(now)
	proposal.IntakeID = created.Session.ID
	ready, err := store.CompleteIntakeWithProposal(ctx, created.Session.ID, 0, proposal, now)
	if err != nil {
		t.Fatalf("CompleteIntakeWithProposal() error = %v", err)
	}
	if ready.Proposal == nil || ready.Proposal.Revision != 1 {
		t.Fatalf("ready = %+v", ready)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	got, found, err := reopened.GetIntake(ctx, created.Session.ID)
	if err != nil || !found {
		t.Fatalf("GetIntake() found=%v err=%v", found, err)
	}
	if got.Proposal == nil || got.Proposal.DesiredState != proposal.DesiredState || len(got.ConversationRefs) != 1 || got.ConversationRefs[0].TurnID != "turn-2" {
		t.Fatalf("restart snapshot = %+v", got)
	}
}

func TestIntakeStoreConfirmationIsAtomicStaleSafeAndIdempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	seedProject(t, store, "confirm-project")
	now := time.Date(2026, 8, 26, 5, 30, 0, 0, time.UTC)
	session := domain.IntakeSession{ID: "intake-confirm", SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "confirm-project", Statement: "Confirm atomically", Status: domain.IntakeStatusCaptured, CreatedAt: now, UpdatedAt: now}
	if _, err := store.CreateIntake(ctx, session, nil, ports.IntakeIdempotency{Key: "capture-confirm", Fingerprint: "fp-capture"}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if _, err := store.BeginIntakeAnalysis(ctx, session.ID, 0, now); err != nil {
		t.Fatalf("begin: %v", err)
	}
	proposal := storeProposal(now)
	proposal.IntakeID = session.ID
	if _, err := store.CompleteIntakeWithProposal(ctx, session.ID, 0, proposal, now); err != nil {
		t.Fatalf("proposal: %v", err)
	}
	space, err := store.EnsureWorkResponsibilitySpace(ctx, "confirm-project")
	if err != nil {
		t.Fatalf("space: %v", err)
	}
	outcome := domain.Outcome{ID: "out-confirm", SpaceID: space.ID, Title: proposal.Title, CurrentRevisionNumber: 1, CreatedAt: now, UpdatedAt: now}
	contract := domain.ContractRevision{ID: "cr-confirm", OutcomeID: outcome.ID, Number: 1, Goal: proposal.DesiredState, SuccessCriteria: []string{"Criterion"}, Criteria: []domain.ContractCriterion{{ID: "crit-confirm", ContractRevisionID: "cr-confirm", Position: 1, Text: "Criterion"}}, Review: "Review", StopConditions: []string{"Stop"}, EvidenceExpectations: []domain.ContractEvidenceExpectation{{CriterionID: "crit-confirm", Descriptions: []string{"Check"}}}, Facets: proposal.Facets, CreatedAt: now}
	request := ports.IntakeIdempotency{Key: "confirm-key", Fingerprint: "confirm-fp"}
	first, err := store.ConfirmIntakeWithOutcome(ctx, session.ID, 1, outcome, contract, request, now)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	replay, err := store.ConfirmIntakeWithOutcome(ctx, session.ID, 1, domain.Outcome{ID: "other"}, domain.ContractRevision{}, request, now)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if first.ConfirmedOutcome == nil || replay.ConfirmedOutcome == nil || replay.ConfirmedOutcome.ID != first.ConfirmedOutcome.ID {
		t.Fatalf("confirmation replay = %+v", replay)
	}
	differentKey, err := store.ConfirmIntakeWithOutcome(ctx, session.ID, 1, domain.Outcome{ID: "other-new-key"}, domain.ContractRevision{}, ports.IntakeIdempotency{Key: "confirm-retry-key", Fingerprint: "retry-fp"}, now)
	if err != nil || differentKey.ConfirmedOutcome == nil || differentKey.ConfirmedOutcome.ID != first.ConfirmedOutcome.ID {
		t.Fatalf("different-key retry = %+v err=%v", differentKey, err)
	}
	if _, found, _ := store.GetOutcome(ctx, "other-new-key"); found {
		t.Fatal("different-key retry created a second Outcome")
	}

	_, err = store.ConfirmIntakeWithOutcome(ctx, session.ID, 0, domain.Outcome{ID: "out-stale"}, domain.ContractRevision{}, ports.IntakeIdempotency{Key: "confirm-stale", Fingerprint: "stale"}, now)
	if err == nil {
		t.Fatal("stale confirmation must fail")
	}
	if _, found, _ := store.GetOutcome(ctx, "out-stale"); found {
		t.Fatal("stale confirmation persisted a partial Outcome")
	}
}

func TestIntakeStoreRecoversInterruptedAnalysisAndPersistsCancellation(t *testing.T) {
	dataDir := t.TempDir()
	store, err := sqlitetest.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, store, "recovery-project")
	now := time.Date(2026, 8, 26, 5, 45, 0, 0, time.UTC)
	session := domain.IntakeSession{ID: "intake-interrupted", SourceSurface: domain.IntakeSourceWork, Purpose: domain.IntakePurposeOutcome, ProjectID: "recovery-project", Statement: "Recover me", Status: domain.IntakeStatusCaptured, CreatedAt: now, UpdatedAt: now}
	if _, err := store.CreateIntake(ctx, session, nil, ports.IntakeIdempotency{Key: "capture-recovery", Fingerprint: "fp"}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if _, err := store.BeginIntakeAnalysis(ctx, session.ID, 0, now); err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	reopened, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	recovered, err := reopened.RecoverInterruptedIntakeAnalyses(ctx, now.Add(time.Minute))
	if err != nil || recovered != 1 {
		t.Fatalf("recover count=%d err=%v", recovered, err)
	}
	failed, found, err := reopened.GetIntake(ctx, session.ID)
	if err != nil || !found || failed.Session.Status != domain.IntakeStatusAnalysisFailed || failed.Session.FailureCode != "INTAKE_ANALYSIS_INTERRUPTED" {
		t.Fatalf("recovered snapshot=%+v found=%v err=%v", failed, found, err)
	}
	cancelled, err := reopened.CancelIntake(ctx, session.ID, 0, "Owner consciously released this", now.Add(2*time.Minute))
	if err != nil || cancelled.Session.Status != domain.IntakeStatusCancelled || cancelled.Session.CancellationReason != "Owner consciously released this" {
		t.Fatalf("cancelled=%+v err=%v", cancelled, err)
	}
	events, err := reopened.EventsAfter(ctx, 0, 100)
	if err != nil {
		t.Fatalf("read CDC events: %v", err)
	}
	updates := 0
	for _, event := range events {
		if event.Type == "intake_updated" {
			updates++
		}
	}
	if updates < 2 {
		t.Fatalf("intake_updated CDC events = %d, want recovery and cancellation trigger events", updates)
	}
}

func TestResponsibilityLinkStoreIsIdempotentAndOneTimeEnd(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	seedProject(t, store, "link-project")
	space, err := store.EnsureWorkResponsibilitySpace(ctx, "link-project")
	if err != nil {
		t.Fatalf("space: %v", err)
	}
	now := time.Date(2026, 8, 26, 6, 0, 0, 0, time.UTC)
	outcome := domain.Outcome{ID: "out-link-store", SpaceID: space.ID, Title: "Linked Outcome", CreatedAt: now, UpdatedAt: now}
	contract := domain.ContractRevision{ID: "cr-link-store", OutcomeID: outcome.ID, Goal: "Linked", SuccessCriteria: []string{"Criterion"}, Review: "Review", CreatedAt: now}
	if err := store.CreateOutcomeWithContract(ctx, outcome, contract, "outcome-link-key"); err != nil {
		t.Fatalf("outcome: %v", err)
	}
	link := domain.ResponsibilityLink{ID: "rlink-store", SourceOpenLoopID: "loop-store", DestinationOutcomeID: outcome.ID, Creator: domain.ResponsibilityLinkCreatorOwner, Reason: "Preserve lineage", CreatedAt: now}
	request := ports.ResponsibilityLinkIdempotency{Key: "rlink-key", Fingerprint: "rlink-fp"}
	first, err := store.CreateResponsibilityLink(ctx, link, request)
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	replay, err := store.CreateResponsibilityLink(ctx, domain.ResponsibilityLink{ID: "different"}, request)
	if err != nil || replay.ID != first.ID {
		t.Fatalf("replay = %+v err=%v", replay, err)
	}
	ended, found, err := store.EndResponsibilityLink(ctx, first.ID, domain.ResponsibilityLinkCreatorOwner, "Owner ended lineage", now.Add(time.Hour))
	if err != nil || !found || ended.EndedAt == nil {
		t.Fatalf("end found=%v link=%+v err=%v", found, ended, err)
	}
	if _, _, err := store.EndResponsibilityLink(ctx, first.ID, domain.ResponsibilityLinkCreatorOwner, "Again", now.Add(2*time.Hour)); err == nil {
		t.Fatal("second end must conflict")
	}
}

func storeProposal(now time.Time) domain.OutcomeContractProposal {
	return domain.OutcomeContractProposal{
		ID: "proposal-1", Revision: 1, Title: "Durable intake", DesiredState: "Intake survives restart.",
		Criteria:     []domain.ProposedCriterion{{ID: "pc-1", Text: "Criterion", EvidenceExpected: []string{"Check"}}},
		ReviewMethod: "Review", StopConditions: []string{"Stop"}, Facets: []domain.ContractFacet{{Kind: domain.ContractFacetSoftware, Summary: "Persistence"}}, CreatedAt: now,
	}
}
