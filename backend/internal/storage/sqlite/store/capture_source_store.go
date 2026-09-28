package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/gen"
)

var _ ports.CaptureSourceStore = (*Store)(nil)

// CreateCaptureGrant inserts one independently governed modality grant.
func (s *Store) CreateCaptureGrant(ctx context.Context, grant domain.CaptureGrant) error {
	if !grant.Modality.K4Writable() || grant.PolicyGeneration != 1 || grant.State == domain.CaptureGrantRevoked || grant.State == domain.CaptureGrantDeleted || grant.Validate() != nil {
		return domain.ErrCaptureInvalid
	}
	p, err := captureGrantInsert(grant)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.InsertCaptureGrant(ctx, p); err != nil {
		if isSQLiteUnique(err) || isSQLitePrimaryKey(err) {
			return domain.ErrCaptureConflict
		}
		return fmt.Errorf("create capture grant: %w", err)
	}
	return nil
}

// GetCaptureGrant reads a grant only within its owner scope.
func (s *Store) GetCaptureGrant(ctx context.Context, ownerID, grantID string) (domain.CaptureGrant, bool, error) {
	row, err := s.qr.GetCaptureGrant(ctx, gen.GetCaptureGrantParams{OwnerID: ownerID, GrantID: grantID})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.CaptureGrant{}, false, nil
	}
	if err != nil {
		return domain.CaptureGrant{}, false, fmt.Errorf("get capture grant: %w", err)
	}
	grant, err := captureGrantFromGen(row)
	return grant, true, err
}

// UpdateCaptureGrantPolicy advances one grant generation by CAS. The single
// SQLite writer serializes it with conditional source inserts. An acknowledged
// update has crossed the final SQL insertion boundary
// of every older write; later writes carrying the old generation insert zero
// rows. A future collector must clean up any staged blob on that zero-row path.
func (s *Store) UpdateCaptureGrantPolicy(ctx context.Context, grant domain.CaptureGrant, expectedGeneration int64) (bool, error) {
	if !grant.Modality.K4Writable() || grant.State == domain.CaptureGrantDeleted || expectedGeneration < 1 || grant.PolicyGeneration != expectedGeneration+1 || grant.Validate() != nil {
		return false, domain.ErrCaptureInvalid
	}
	p, err := captureGrantInsert(grant)
	if err != nil {
		return false, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	current, err := s.qw.GetCaptureGrant(ctx, gen.GetCaptureGrantParams{OwnerID: grant.OwnerID, GrantID: grant.GrantID})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read capture grant for CAS: %w", err)
	}
	if current.PolicyGeneration != expectedGeneration || current.Modality != string(grant.Modality) || current.DeviceID != grant.DeviceID || current.OsPermissionKind != grant.OSPermissionKind || !current.CreatedAt.Equal(grant.CreatedAt) || grant.UpdatedAt.Before(current.UpdatedAt) || current.State == string(domain.CaptureGrantDeleted) || current.State == string(domain.CaptureGrantRevoked) && grant.State != domain.CaptureGrantDeleted {
		return false, nil
	}
	n, err := s.qw.UpdateCaptureGrantPolicy(ctx, gen.UpdateCaptureGrantPolicyParams{
		OsPermissionState: p.OsPermissionState, ScopeJson: p.ScopeJson, Purpose: p.Purpose,
		AllowedSpaceIdsJson: p.AllowedSpaceIdsJson, DisclosurePolicy: p.DisclosurePolicy,
		RawRetentionSeconds: p.RawRetentionSeconds, DerivedRetentionSeconds: p.DerivedRetentionSeconds,
		Sensitivity: p.Sensitivity, BystanderPolicy: p.BystanderPolicy,
		ExportBehavior: p.ExportBehavior, DeleteBehavior: p.DeleteBehavior, State: p.State,
		UpdatedAt: p.UpdatedAt, PausedAt: p.PausedAt, RevokedAt: p.RevokedAt, DeletedAt: p.DeletedAt,
		OwnerID: p.OwnerID, GrantID: p.GrantID, Modality: p.Modality, DeviceID: p.DeviceID,
		ExpectedGeneration: expectedGeneration,
	})
	if err != nil {
		return false, fmt.Errorf("update capture grant policy: %w", err)
	}
	return n == 1, nil
}

// CreateSourceArtifact inserts metadata only when its grant generation is current.
func (s *Store) CreateSourceArtifact(ctx context.Context, a domain.SourceArtifact) error {
	if !a.Modality.K4Writable() || a.LifecycleState != "available" || a.DeletedAt != nil || a.Validate() != nil {
		return domain.ErrCaptureInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.InsertSourceArtifactIfGrantCurrent(ctx, gen.InsertSourceArtifactIfGrantCurrentParams{
		ArtifactID: a.ArtifactID, OwnerID: a.OwnerID, GrantID: a.GrantID,
		GrantGeneration: a.GrantGeneration, Modality: string(a.Modality), ContentKind: a.ContentKind,
		MimeType: a.MIMEType, HashAlgorithm: a.HashAlgorithm, ContentHash: a.ContentHash,
		BlobCustodyRef: a.BlobCustodyRef, ProvenanceKind: a.ProvenanceKind, ProvenanceRef: a.ProvenanceRef,
		CapturedAt: a.CapturedAt.UTC(), LifecycleState: a.LifecycleState, CreatedAt: a.CreatedAt.UTC(),
	})
	if err != nil {
		if isSQLiteUnique(err) || isSQLitePrimaryKey(err) {
			return domain.ErrCaptureConflict
		}
		return fmt.Errorf("create source artifact: %w", err)
	}
	if n != 1 {
		return domain.ErrCaptureStaleGrant
	}
	return nil
}

// GetSourceArtifact hides source custody when its grant is no longer current.
func (s *Store) GetSourceArtifact(ctx context.Context, ownerID, artifactID string) (domain.SourceArtifact, bool, error) {
	row, err := s.qr.GetSourceArtifact(ctx, gen.GetSourceArtifactParams{OwnerID: ownerID, ArtifactID: artifactID})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SourceArtifact{}, false, nil
	}
	if err != nil {
		return domain.SourceArtifact{}, false, fmt.Errorf("get source artifact: %w", err)
	}
	a := domain.SourceArtifact{
		ArtifactID: row.ArtifactID, OwnerID: row.OwnerID, GrantID: row.GrantID,
		GrantGeneration: row.GrantGeneration, Modality: domain.CaptureModality(row.Modality),
		ContentKind: row.ContentKind, MIMEType: row.MimeType, HashAlgorithm: row.HashAlgorithm,
		ContentHash: row.ContentHash, BlobCustodyRef: row.BlobCustodyRef, ProvenanceKind: row.ProvenanceKind,
		ProvenanceRef: row.ProvenanceRef, CapturedAt: row.CapturedAt.UTC(), LifecycleState: row.LifecycleState,
		CreatedAt: row.CreatedAt.UTC(), DeletedAt: nullTimeToPtr(row.DeletedAt),
	}
	return a, true, a.Validate()
}

// CreateSourceSegment inserts lineage only under a current, active grant.
func (s *Store) CreateSourceSegment(ctx context.Context, segment domain.SourceSegment) error {
	if !segment.Modality.K4Writable() || segment.LifecycleState != "available" || segment.DeletedAt != nil || segment.Validate() != nil {
		return domain.ErrCaptureInvalid
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	n, err := s.qw.InsertSourceSegmentIfGrantCurrent(ctx, gen.InsertSourceSegmentIfGrantCurrentParams{
		SegmentID: segment.SegmentID, OwnerID: segment.OwnerID, GrantID: segment.GrantID,
		ArtifactID: segment.ArtifactID, GrantGeneration: segment.GrantGeneration,
		Modality: string(segment.Modality), Ordinal: segment.Ordinal,
		StartOffset: nullInt64(segment.StartOffset), EndOffset: nullInt64(segment.EndOffset),
		StartAt: timePtrToNull(segment.StartAt), EndAt: timePtrToNull(segment.EndAt),
		ProvenanceRef: segment.ProvenanceRef, LifecycleState: segment.LifecycleState, CreatedAt: segment.CreatedAt.UTC(),
	})
	if err != nil {
		if isSQLiteUnique(err) || isSQLitePrimaryKey(err) {
			return domain.ErrCaptureConflict
		}
		return fmt.Errorf("create source segment: %w", err)
	}
	if n != 1 {
		return domain.ErrCaptureStaleGrant
	}
	return nil
}

// GetSourceSegment hides segment lineage when its grant is no longer current.
func (s *Store) GetSourceSegment(ctx context.Context, ownerID, segmentID string) (domain.SourceSegment, bool, error) {
	row, err := s.qr.GetSourceSegment(ctx, gen.GetSourceSegmentParams{OwnerID: ownerID, SegmentID: segmentID})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SourceSegment{}, false, nil
	}
	if err != nil {
		return domain.SourceSegment{}, false, fmt.Errorf("get source segment: %w", err)
	}
	seg := domain.SourceSegment{
		SegmentID: row.SegmentID, OwnerID: row.OwnerID, GrantID: row.GrantID, ArtifactID: row.ArtifactID,
		GrantGeneration: row.GrantGeneration, Modality: domain.CaptureModality(row.Modality), Ordinal: row.Ordinal,
		StartOffset: int64Ptr(row.StartOffset), EndOffset: int64Ptr(row.EndOffset),
		StartAt: nullTimeToPtr(row.StartAt), EndAt: nullTimeToPtr(row.EndAt),
		ProvenanceRef: row.ProvenanceRef, LifecycleState: row.LifecycleState,
		CreatedAt: row.CreatedAt.UTC(), DeletedAt: nullTimeToPtr(row.DeletedAt),
	}
	return seg, true, seg.Validate()
}

func captureGrantInsert(g domain.CaptureGrant) (gen.InsertCaptureGrantParams, error) {
	scope, err := json.Marshal(g.Scope)
	if err != nil {
		return gen.InsertCaptureGrantParams{}, domain.ErrCaptureInvalid
	}
	spaces := g.AllowedSpaceIDs
	if spaces == nil {
		spaces = []string{}
	}
	allowed, err := json.Marshal(spaces)
	if err != nil {
		return gen.InsertCaptureGrantParams{}, domain.ErrCaptureInvalid
	}
	return gen.InsertCaptureGrantParams{
		GrantID: g.GrantID, OwnerID: g.OwnerID, DeviceID: g.DeviceID, Modality: string(g.Modality),
		OsPermissionKind: g.OSPermissionKind, OsPermissionState: g.OSPermissionState,
		ScopeJson: string(scope), Purpose: g.Purpose, AllowedSpaceIdsJson: string(allowed),
		ProcessingRoute: g.ProcessingRoute, DisclosurePolicy: g.DisclosurePolicy,
		RawRetentionSeconds: g.RawRetentionSeconds, DerivedRetentionSeconds: g.DerivedRetentionSeconds,
		Sensitivity: g.Sensitivity, BystanderPolicy: g.BystanderPolicy, ExportBehavior: g.ExportBehavior,
		DeleteBehavior: g.DeleteBehavior, State: string(g.State), PolicyGeneration: g.PolicyGeneration,
		CreatedAt: g.CreatedAt.UTC(), UpdatedAt: g.UpdatedAt.UTC(), PausedAt: timePtrToNull(g.PausedAt),
		RevokedAt: timePtrToNull(g.RevokedAt), DeletedAt: timePtrToNull(g.DeletedAt),
	}, nil
}

func captureGrantFromGen(row gen.CaptureGrant) (domain.CaptureGrant, error) {
	var scope domain.CaptureScope
	var spaces []string
	if err := json.Unmarshal([]byte(row.ScopeJson), &scope); err != nil {
		return domain.CaptureGrant{}, fmt.Errorf("decode capture scope: %w", err)
	}
	if err := json.Unmarshal([]byte(row.AllowedSpaceIdsJson), &spaces); err != nil {
		return domain.CaptureGrant{}, fmt.Errorf("decode capture spaces: %w", err)
	}
	g := domain.CaptureGrant{
		GrantID: row.GrantID, OwnerID: row.OwnerID, DeviceID: row.DeviceID,
		Modality: domain.CaptureModality(row.Modality), OSPermissionKind: row.OsPermissionKind,
		OSPermissionState: row.OsPermissionState, Scope: scope, Purpose: row.Purpose,
		AllowedSpaceIDs: spaces, ProcessingRoute: row.ProcessingRoute, DisclosurePolicy: row.DisclosurePolicy,
		RawRetentionSeconds: row.RawRetentionSeconds, DerivedRetentionSeconds: row.DerivedRetentionSeconds,
		Sensitivity: row.Sensitivity, BystanderPolicy: row.BystanderPolicy, ExportBehavior: row.ExportBehavior,
		DeleteBehavior: row.DeleteBehavior, State: domain.CaptureGrantState(row.State),
		PolicyGeneration: row.PolicyGeneration, CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
		PausedAt: nullTimeToPtr(row.PausedAt), RevokedAt: nullTimeToPtr(row.RevokedAt), DeletedAt: nullTimeToPtr(row.DeletedAt),
	}
	return g, g.Validate()
}
