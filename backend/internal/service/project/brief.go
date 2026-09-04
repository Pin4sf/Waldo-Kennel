package project

import (
	"context"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd/apierr"
)

// BriefStore is the narrow durable boundary for immutable Project Brief
// revisions. It is separate from Store so existing Project registry consumers
// are not forced to know about the Slice 2 context surface.
type BriefStore interface {
	AppendProjectBriefRevision(ctx context.Context, projectID domain.ProjectID, expectedCurrent int64, revision domain.ProjectBriefRevision) (domain.ProjectBriefRevision, bool, error)
	GetCurrentProjectBriefRevision(ctx context.Context, projectID domain.ProjectID) (domain.ProjectBriefRevision, bool, error)
	ListProjectBriefRevisions(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectBriefRevision, error)
}

// BriefManager is the daemon-facing Project Brief use-case surface.
type BriefManager interface {
	CurrentBrief(ctx context.Context, projectID domain.ProjectID) (domain.ProjectBriefRevision, bool, error)
	BriefHistory(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectBriefRevision, error)
	ReviseBrief(ctx context.Context, projectID domain.ProjectID, in ReviseBriefInput) (domain.ProjectBriefRevision, error)
}

// ReviseBriefInput is a complete replacement snapshot for the next immutable
// Project Brief revision. ExpectedCurrentRevisionNumber is an optimistic
// concurrency guard; zero means the caller expects no Brief yet.
type ReviseBriefInput struct {
	ExpectedCurrentRevisionNumber int64
	Purpose                       string
	ProductContext                string
	TechnicalContext              string
	ArchitectureSummary           string
	Conventions                   []string
	Constraints                   []string
	SetupExpectations             string
	RunExpectations               string
	TestExpectations              string
	Provenance                    []string
}

var _ BriefManager = (*Service)(nil)

// CurrentBrief returns the current immutable Project Brief revision. ok=false
// means the Project exists but has no Brief yet.
func (m *Service) CurrentBrief(ctx context.Context, projectID domain.ProjectID) (domain.ProjectBriefRevision, bool, error) {
	if err := m.requireActiveBriefProject(ctx, projectID); err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	store, err := m.briefStore()
	if err != nil {
		return domain.ProjectBriefRevision{}, false, err
	}
	revision, ok, err := store.GetCurrentProjectBriefRevision(ctx, projectID)
	if err != nil {
		return domain.ProjectBriefRevision{}, false, apierr.Internal("PROJECT_BRIEF_LOAD_FAILED", "Failed to load Project Brief")
	}
	return revision, ok, nil
}

// BriefHistory returns immutable history in ascending revision order.
func (m *Service) BriefHistory(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectBriefRevision, error) {
	if err := m.requireActiveBriefProject(ctx, projectID); err != nil {
		return nil, err
	}
	store, err := m.briefStore()
	if err != nil {
		return nil, err
	}
	revisions, err := store.ListProjectBriefRevisions(ctx, projectID)
	if err != nil {
		return nil, apierr.Internal("PROJECT_BRIEF_LOAD_FAILED", "Failed to load Project Brief history")
	}
	return revisions, nil
}

// ReviseBrief appends a complete immutable snapshot and atomically advances the
// current projection. It does not mutate Outcomes, Contracts, Plans, proof, or
// execution state.
func (m *Service) ReviseBrief(ctx context.Context, projectID domain.ProjectID, in ReviseBriefInput) (domain.ProjectBriefRevision, error) {
	if err := m.requireActiveBriefProject(ctx, projectID); err != nil {
		return domain.ProjectBriefRevision{}, err
	}
	if in.ExpectedCurrentRevisionNumber < 0 {
		return domain.ProjectBriefRevision{}, apierr.Invalid(
			"PROJECT_BRIEF_EXPECTED_REVISION_INVALID",
			"Expected Project Brief revision must not be negative",
			nil,
		)
	}

	revision := domain.ProjectBriefRevision{
		ID:                  domain.ProjectBriefRevisionID("pbr-" + uuid.NewString()),
		ProjectID:           projectID,
		Number:              in.ExpectedCurrentRevisionNumber + 1,
		Purpose:             in.Purpose,
		ProductContext:      in.ProductContext,
		TechnicalContext:    in.TechnicalContext,
		ArchitectureSummary: in.ArchitectureSummary,
		Conventions:         append([]string(nil), in.Conventions...),
		Constraints:         append([]string(nil), in.Constraints...),
		SetupExpectations:   in.SetupExpectations,
		RunExpectations:     in.RunExpectations,
		TestExpectations:    in.TestExpectations,
		Provenance:          append([]string(nil), in.Provenance...),
		CreatedAt:           m.clock().UTC(),
	}
	if err := revision.Validate(); err != nil {
		return domain.ProjectBriefRevision{}, apierr.Invalid("PROJECT_BRIEF_INVALID", err.Error(), nil)
	}

	store, err := m.briefStore()
	if err != nil {
		return domain.ProjectBriefRevision{}, err
	}
	persisted, conflict, err := store.AppendProjectBriefRevision(ctx, projectID, in.ExpectedCurrentRevisionNumber, revision)
	if err != nil {
		return domain.ProjectBriefRevision{}, apierr.Internal("PROJECT_BRIEF_WRITE_FAILED", "Failed to revise Project Brief")
	}
	if conflict {
		current := int64(0)
		if existing, ok, loadErr := store.GetCurrentProjectBriefRevision(ctx, projectID); loadErr == nil && ok {
			current = existing.Number
		}
		return domain.ProjectBriefRevision{}, apierr.Conflict(
			"PROJECT_BRIEF_REVISION_CONFLICT",
			"Project Brief changed since it was read",
			map[string]any{
				"expectedRevision": in.ExpectedCurrentRevisionNumber,
				"currentRevision":  current,
			},
		)
	}
	return persisted, nil
}

func (m *Service) briefStore() (BriefStore, error) {
	store, ok := m.store.(BriefStore)
	if !ok || store == nil {
		return nil, apierr.Internal("PROJECT_BRIEF_STORE_UNAVAILABLE", "Project Brief storage is unavailable")
	}
	return store, nil
}

func (m *Service) requireActiveBriefProject(ctx context.Context, projectID domain.ProjectID) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	row, ok, err := m.store.GetProject(ctx, string(projectID))
	if err != nil {
		return apierr.Internal("PROJECT_LOAD_FAILED", "Failed to load project")
	}
	if !ok || !row.ArchivedAt.IsZero() {
		return apierr.NotFound("PROJECT_NOT_FOUND", "Unknown project")
	}
	return nil
}
