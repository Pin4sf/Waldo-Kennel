package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/sqlitetest"
)

// A contract revision's authority ceiling, stop conditions, expected evidence,
// temporal condition, and facets live in a side relation (migration 0104).
// They were written only by intake confirmation and read back only by the
// intake snapshot path, so every OTHER creator dropped them on write and every
// other reader saw an all-false ceiling with no stop conditions.
//
// That is not cosmetic: OverClaimedAuthority and AuthorityWidenings compare a
// contributor's claim against the parent ceiling this read returns, so the
// gates that are supposed to refuse a widened claim were comparing against a
// ceiling that had never been loaded.
func TestContractRevisionCoreSurvivesEveryCreatorAndReader(t *testing.T) {
	s, err := sqlitetest.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	ctx := context.Background()
	seedProject(t, s, "core-project")
	space, err := s.EnsureWorkResponsibilitySpace(ctx, "core-project")
	if err != nil {
		t.Fatalf("ensure work space: %v", err)
	}
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	temporal := "Before the end of the local calendar day."
	revisionID := domain.ContractRevisionID("cr-core-1")
	ceiling := domain.ProposedAuthority{ReadWorkspace: true, WriteWorkspace: true, ExecuteLocal: true}
	revision := domain.ContractRevision{
		ID:              revisionID,
		OutcomeID:       "out-core",
		Goal:            "The ceiling survives a round trip.",
		SuccessCriteria: []string{"It survives."},
		Criteria: []domain.ContractCriterion{
			{ID: "crit-core", ContractRevisionID: revisionID, Position: 1, Text: "It survives."},
		},
		Review:               "Read it back.",
		AuthorityCeiling:     ceiling,
		StopConditions:       []string{"Stop before any external effect."},
		EvidenceExpectations: []domain.ContractEvidenceExpectation{{CriterionID: "crit-core", Descriptions: []string{"A read-back shows it."}}},
		TemporalCondition:    &temporal,
		Facets:               []domain.ContractFacet{{Kind: domain.ContractFacetSoftware, Summary: "Storage fix"}},
		CreatedAt:            now,
	}

	// CreateOutcomeWithContract is one of the creators that never wrote the
	// side relation; intake confirmation was the only one that did.
	outcome := domain.Outcome{ID: "out-core", SpaceID: space.ID, Title: "Core survives"}
	if err := s.CreateOutcomeWithContract(ctx, outcome, revision, "core-key"); err != nil {
		t.Fatalf("create outcome: %v", err)
	}

	history, err := s.ListContractRevisions(ctx, outcome.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one revision, got %d", len(history))
	}
	got := history[0]
	if got.AuthorityCeiling != ceiling {
		t.Errorf("authority ceiling lost: want %+v got %+v", ceiling, got.AuthorityCeiling)
	}
	if len(got.StopConditions) != 1 || got.StopConditions[0] != "Stop before any external effect." {
		t.Errorf("stop conditions lost: %+v", got.StopConditions)
	}
	if len(got.EvidenceExpectations) != 1 || len(got.EvidenceExpectations[0].Descriptions) != 1 {
		t.Errorf("evidence expectations lost: %+v", got.EvidenceExpectations)
	}
	if got.TemporalCondition == nil || *got.TemporalCondition != temporal {
		t.Errorf("temporal condition lost: %v", got.TemporalCondition)
	}
	if len(got.Facets) != 1 || got.Facets[0].Kind != domain.ContractFacetSoftware {
		t.Errorf("facets lost: %+v", got.Facets)
	}

	// An appended correction is the other path a ceiling can arrive by, and it
	// has to survive the same way — a correction that silently reset authority
	// to nothing would be worse than one that refused.
	narrowed := domain.ProposedAuthority{ReadWorkspace: true}
	next := revision
	next.ID = domain.ContractRevisionID("cr-core-2")
	next.Criteria = []domain.ContractCriterion{{ID: "crit-core-2", ContractRevisionID: "cr-core-2", Position: 1, Text: "It survives."}}
	next.AuthorityCeiling = narrowed
	next.StopConditions = []string{"Stop on the first refusal."}
	next.EvidenceExpectations = []domain.ContractEvidenceExpectation{{CriterionID: "crit-core-2", Descriptions: []string{"A read-back shows it."}}}
	if _, err := s.AppendContractRevision(ctx, outcome.ID, 1, next); err != nil {
		t.Fatalf("append revision: %v", err)
	}
	history, err = s.ListContractRevisions(ctx, outcome.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("list revisions after append: history=%d err=%v", len(history), err)
	}
	if history[1].AuthorityCeiling != narrowed {
		t.Errorf("appended ceiling lost: want %+v got %+v", narrowed, history[1].AuthorityCeiling)
	}
	if len(history[1].StopConditions) != 1 || history[1].StopConditions[0] != "Stop on the first refusal." {
		t.Errorf("appended stop conditions lost: %+v", history[1].StopConditions)
	}
	// The earlier revision is immutable and keeps its own wider ceiling.
	if history[0].AuthorityCeiling != ceiling {
		t.Errorf("append rewrote an earlier revision's ceiling: %+v", history[0].AuthorityCeiling)
	}
}
