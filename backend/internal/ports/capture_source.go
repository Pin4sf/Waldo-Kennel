package ports

import (
	"context"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// CaptureSourceStore persists consent policy and untrusted source lineage.
// Every write is owner-scoped. The source insert methods are final durable
// generation fences; no collector or blob-cleanup adapter exists in K4-1.
type CaptureSourceStore interface {
	CreateCaptureGrant(ctx context.Context, grant domain.CaptureGrant) error
	GetCaptureGrant(ctx context.Context, ownerID, grantID string) (domain.CaptureGrant, bool, error)
	// UpdateCaptureGrantPolicy is a CAS on expectedGeneration. Any policy or
	// state change advances it exactly once. Deleted is reserved until a later
	// cleanup worker can prove raw and derived custody has been removed.
	UpdateCaptureGrantPolicy(ctx context.Context, grant domain.CaptureGrant, expectedGeneration int64) (bool, error)
	CreateSourceArtifact(ctx context.Context, artifact domain.SourceArtifact) error
	GetSourceArtifact(ctx context.Context, ownerID, artifactID string) (domain.SourceArtifact, bool, error)
	CreateSourceSegment(ctx context.Context, segment domain.SourceSegment) error
	GetSourceSegment(ctx context.Context, ownerID, segmentID string) (domain.SourceSegment, bool, error)
}
