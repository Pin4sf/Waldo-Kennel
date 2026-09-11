package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func planningSessionFixture(revision domain.ContractRevision, now time.Time) domain.PlanningSession {
	contextJSON := []byte(`{"root":"/tmp/provider-project","files":[],"digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	return domain.PlanningSession{
		ID: "planning-store", OutcomeID: revision.OutcomeID, ProjectID: "provider-project",
		ContractRevisionID: revision.ID, ContractRevisionNumber: revision.Number, Revision: 1,
		Status: domain.PlanningSessionActive, WaitingOn: domain.PlanningWaitingOwner,
		Binding:             domain.PlanningBinding{Mode: domain.PlanningModeDirectAPI, Provider: "openai", ModelSelection: domain.PlanningModelExplicit, Model: "planner-test"},
		ContextMode:         domain.PlanningContextRepositoryRead,
		PlanningGrantDigest: domain.DigestSHA256([]byte("repository-read-only")), ContextDigest: domain.DigestSHA256(contextJSON), ContextSnapshotJSON: contextJSON,
		RequestKey: "planning-start-key", RequestFingerprint: domain.DigestSHA256([]byte("planning-start")), CreatedAt: now, UpdatedAt: now,
	}
}

func TestPlanningSessionStore_IdempotentTurnsAndCanonicalPlanLink(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	revision := seedProviderPlanOutcome(t, s)
	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	session := planningSessionFixture(revision, now)

	created, replay, err := s.CreatePlanningSession(ctx, session)
	if err != nil || replay {
		t.Fatalf("create planning session replay=%v err=%v", replay, err)
	}
	replayed, replay, err := s.CreatePlanningSession(ctx, session)
	if err != nil || !replay || replayed.ID != created.ID {
		t.Fatalf("replay planning session = %+v replay=%v err=%v", replayed, replay, err)
	}

	owner := domain.PlanningTurn{
		ID: "planning-turn-owner", Role: domain.PlanningTurnOwner, Kind: domain.PlanningTurnFinalizeRequest,
		Text: "Propose the plan now.", RequestKey: "planning-finalize-key",
		RequestFingerprint: domain.DigestSHA256([]byte("planning-finalize")), CreatedAt: now.Add(time.Second),
	}
	current, storedOwner, replay, err := s.AppendPlanningOwnerTurn(ctx, session.ID, 1, owner)
	if err != nil || replay || current.Revision != 2 || current.WaitingOn != domain.PlanningWaitingProvider || storedOwner.Sequence != 1 {
		t.Fatalf("append owner turn session=%+v turn=%+v replay=%v err=%v", current, storedOwner, replay, err)
	}
	current, replayedOwner, replay, err := s.AppendPlanningOwnerTurn(ctx, session.ID, 1, owner)
	if err != nil || !replay || replayedOwner.ID != storedOwner.ID || current.Revision != 2 {
		t.Fatalf("replay owner turn session=%+v turn=%+v replay=%v err=%v", current, replayedOwner, replay, err)
	}

	run := domain.IntelligenceRun{
		ID: "intel-planning-store", Kind: domain.IntelligenceRunPlanDraft, ProjectID: session.ProjectID,
		OutcomeID: revision.OutcomeID, ContractRevisionID: revision.ID, SourceRevision: revision.Number,
		RequestedProvider: session.Binding.Provider, RequestedModel: session.Binding.Model,
		InputDigest: domain.DigestSHA256([]byte("planning turn input")), Status: domain.IntelligenceRunRunning, CreatedAt: now.Add(2 * time.Second),
	}
	if err := s.CreateIntelligenceRun(ctx, run); err != nil {
		t.Fatalf("create planning intelligence run: %v", err)
	}
	planner := domain.PlanningTurn{
		ID: "planning-turn-planner", ReplyToTurnID: storedOwner.ID, Role: domain.PlanningTurnPlanner,
		Kind: domain.PlanningTurnPlanProposal, Text: "A two-step Plan is ready for review.",
		StructuredPayload: []byte(`{"Kind":"plan_proposal","Message":"A two-step Plan is ready for review."}`),
		IntelligenceRunID: run.ID, CreatedAt: now.Add(3 * time.Second),
	}
	current, err = s.AppendPlanningProviderTurn(ctx, session.ID, 2, planner, "openai", "planner-test", "")
	if err != nil || current.Revision != 3 || current.WaitingOn != domain.PlanningWaitingOwner {
		t.Fatalf("append planner turn session=%+v err=%v", current, err)
	}

	plan := canonicalGraphPlan(t, revision)
	plan.ID = "plan-from-planning-store"
	plan.PlanningSessionID = session.ID
	plan.SourceIntelligenceRunID = run.ID
	saved, err := s.AppendPlanRevision(ctx, revision.OutcomeID, plan)
	if err != nil {
		t.Fatalf("append planning-sourced Plan: %v", err)
	}
	current, err = s.LinkPlanningSessionPlan(ctx, session.ID, 3, saved.ID, run.ID)
	if err != nil {
		t.Fatalf("link planning Plan: %v", err)
	}
	if current.Status != domain.PlanningSessionProposalReady || current.ProposedPlanRevisionID != saved.ID || current.WaitingOn != domain.PlanningWaitingNone {
		t.Fatalf("proposal-ready session = %+v", current)
	}
	got, found, err := s.GetPlanRevisionByPlanningSession(ctx, revision.OutcomeID, session.ID)
	if err != nil || !found || got.PlanningSessionID != session.ID || got.SourceIntelligenceRunID != run.ID {
		t.Fatalf("planning Plan found=%v plan=%+v err=%v", found, got, err)
	}
}

func TestPlanningSessionStore_RejectsChangedRequestAndStaleRevision(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	revision := seedProviderPlanOutcome(t, s)
	now := time.Date(2026, 9, 11, 11, 0, 0, 0, time.UTC)
	session := planningSessionFixture(revision, now)
	if _, _, err := s.CreatePlanningSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	changed := session
	changed.RequestFingerprint = domain.DigestSHA256([]byte("changed semantics"))
	_, _, err := s.CreatePlanningSession(ctx, changed)
	var requestConflict *ports.PlanningRequestConflictError
	if !errors.As(err, &requestConflict) {
		t.Fatalf("changed request error = %v", err)
	}

	owner := domain.PlanningTurn{ID: "stale-owner", Role: domain.PlanningTurnOwner, Kind: domain.PlanningTurnMessage, Text: "Inspect first.", RequestKey: "stale-owner-key", RequestFingerprint: domain.DigestSHA256([]byte("stale-owner")), CreatedAt: now.Add(time.Second)}
	_, _, _, err = s.AppendPlanningOwnerTurn(ctx, session.ID, 99, owner)
	var revisionConflict *ports.PlanningSessionRevisionConflictError
	if !errors.As(err, &revisionConflict) || revisionConflict.Current != 1 {
		t.Fatalf("stale revision error = %v", err)
	}
}
