package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite/sqlitetest"
)

func captureTestGrant(now time.Time, id string, modality domain.CaptureModality) domain.CaptureGrant {
	kind := "screen_recording"
	if modality == domain.CaptureAXTree {
		kind = "accessibility"
	}
	return domain.CaptureGrant{
		GrantID: id, OwnerID: "owner-1", DeviceID: "device-1", Modality: modality,
		OSPermissionKind: kind, OSPermissionState: "granted",
		Scope:   domain.CaptureScope{ExcludeApps: []string{"com.example.vault"}},
		Purpose: "local context", AllowedSpaceIDs: []string{"space-1"},
		ProcessingRoute: "local", DisclosurePolicy: "local_only",
		RawRetentionSeconds: 3600, DerivedRetentionSeconds: 86400,
		Sensitivity: "sensitive", BystanderPolicy: "exclude",
		ExportBehavior: "owner_request_only", DeleteBehavior: "cascade_local_custody",
		State: domain.CaptureGrantActive, PolicyGeneration: 1,
		CreatedAt: now, UpdatedAt: now,
	}
}

func captureTestArtifact(now time.Time, grant domain.CaptureGrant, id string) domain.SourceArtifact {
	kind, mime, provenance := "image", "image/jpeg", "screen_capture"
	if grant.Modality == domain.CaptureAXTree {
		kind, mime, provenance = "structured", "application/json", "accessibility_snapshot"
	}
	return domain.SourceArtifact{
		ArtifactID: id, OwnerID: grant.OwnerID, GrantID: grant.GrantID,
		GrantGeneration: grant.PolicyGeneration, Modality: grant.Modality,
		ContentKind: kind, MIMEType: mime, HashAlgorithm: "sha256",
		ContentHash: strings.Repeat("a", 64), BlobCustodyRef: "local:" + id,
		ProvenanceKind: provenance, ProvenanceRef: "device/source/" + id,
		CapturedAt: now, LifecycleState: "available", CreatedAt: now,
	}
}

func TestCaptureSourceStoreGrantIsolationAndShapes(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	screen := captureTestGrant(now, "screen-grant", domain.CaptureScreenFrame)
	ax := captureTestGrant(now, "ax-grant", domain.CaptureAXTree)
	if err := s.CreateCaptureGrant(ctx, screen); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCaptureGrant(ctx, ax); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCaptureGrant(ctx, screen); !errors.Is(err, domain.ErrCaptureConflict) {
		t.Fatalf("duplicate grant: %v", err)
	}
	if _, found, err := s.GetCaptureGrant(ctx, "other-owner", screen.GrantID); err != nil || found {
		t.Fatalf("cross-owner grant read: found=%v err=%v", found, err)
	}
	gotAX, found, err := s.GetCaptureGrant(ctx, ax.OwnerID, ax.GrantID)
	if err != nil || !found || gotAX.OSPermissionKind != "accessibility" || gotAX.Scope.ExcludeApps[0] != "com.example.vault" {
		t.Fatalf("AX grant read: found=%v grant=%+v err=%v", found, gotAX, err)
	}
	frame := captureTestArtifact(now, screen, "frame-1")
	tree := captureTestArtifact(now, ax, "tree-1")
	if err := s.CreateSourceArtifact(ctx, frame); err != nil {
		t.Fatalf("JPEG frame: %v", err)
	}
	if err := s.CreateSourceArtifact(ctx, tree); err != nil {
		t.Fatalf("structured AX tree: %v", err)
	}
	if err := s.CreateSourceArtifact(ctx, frame); !errors.Is(err, domain.ErrCaptureConflict) {
		t.Fatalf("duplicate artifact: %v", err)
	}
	if _, found, err := s.GetSourceArtifact(ctx, "other-owner", frame.ArtifactID); err != nil || found {
		t.Fatalf("cross-owner artifact read: found=%v err=%v", found, err)
	}
	storedTree, found, err := s.GetSourceArtifact(ctx, ax.OwnerID, tree.ArtifactID)
	if err != nil || !found || storedTree.ContentKind != "structured" || storedTree.MIMEType != "application/json" {
		t.Fatalf("AX artifact read: found=%v artifact=%+v err=%v", found, storedTree, err)
	}
	start, end := int64(0), int64(100)
	seg := domain.SourceSegment{
		SegmentID: "seg-1", OwnerID: ax.OwnerID, GrantID: ax.GrantID, ArtifactID: tree.ArtifactID,
		GrantGeneration: 1, Modality: domain.CaptureAXTree, Ordinal: 0,
		StartOffset: &start, EndOffset: &end, ProvenanceRef: "device/segment/0",
		LifecycleState: "available", CreatedAt: now,
	}
	if err := s.CreateSourceSegment(ctx, seg); err != nil {
		t.Fatalf("AX segment: %v", err)
	}
	frameEnd := now.Add(time.Second)
	frameSegment := domain.SourceSegment{
		SegmentID: "frame-window", OwnerID: screen.OwnerID, GrantID: screen.GrantID,
		ArtifactID: frame.ArtifactID, GrantGeneration: 1, Modality: domain.CaptureScreenFrame,
		Ordinal: 0, StartAt: &now, EndAt: &frameEnd, ProvenanceRef: "device/frame/window",
		LifecycleState: "available", CreatedAt: now,
	}
	if err := s.CreateSourceSegment(ctx, frameSegment); err != nil {
		t.Fatalf("frame time-range segment: %v", err)
	}
	storedSeg, found, err := s.GetSourceSegment(ctx, ax.OwnerID, seg.SegmentID)
	if err != nil || !found || storedSeg.EndOffset == nil || *storedSeg.EndOffset != 100 {
		t.Fatalf("AX segment read: found=%v segment=%+v err=%v", found, storedSeg, err)
	}
	seg.SegmentID, seg.GrantID = "seg-wrong", screen.GrantID
	if err := s.CreateSourceSegment(ctx, seg); !errors.Is(err, domain.ErrCaptureStaleGrant) {
		t.Fatalf("cross-grant segment must reject: %v", err)
	}
	if _, found, err := s.GetSourceSegment(ctx, "other-owner", "seg-1"); err != nil || found {
		t.Fatalf("cross-owner segment read: found=%v err=%v", found, err)
	}
}

func TestCaptureSourceStorePolicyGenerationFencesWrites(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	screen := captureTestGrant(now, "screen", domain.CaptureScreenFrame)
	ax := captureTestGrant(now, "ax", domain.CaptureAXTree)
	for _, g := range []domain.CaptureGrant{screen, ax} {
		if err := s.CreateCaptureGrant(ctx, g); err != nil {
			t.Fatal(err)
		}
	}
	old := captureTestArtifact(now, screen, "before-pause")
	if err := s.CreateSourceArtifact(ctx, old); err != nil {
		t.Fatal(err)
	}
	screen.State = domain.CaptureGrantPaused
	screen.PausedAt = timePtr(now.Add(time.Minute))
	screen.UpdatedAt = now.Add(time.Minute)
	screen.PolicyGeneration = 2
	screen.Scope.ExcludeApps = append(screen.Scope.ExcludeApps, "com.example.private")
	if won, err := s.UpdateCaptureGrantPolicy(ctx, screen, 1); err != nil || !won {
		t.Fatalf("pause policy CAS: won=%v err=%v", won, err)
	}
	if _, found, err := s.GetSourceArtifact(ctx, screen.OwnerID, old.ArtifactID); err != nil || found {
		t.Fatalf("paused grant exposed old custody ref: found=%v err=%v", found, err)
	}
	if won, err := s.UpdateCaptureGrantPolicy(ctx, screen, 1); err != nil || won {
		t.Fatalf("stale policy CAS: won=%v err=%v", won, err)
	}
	stale := captureTestArtifact(now, screen, "after-pause")
	stale.ContentHash = strings.Repeat("b", 64)
	stale.GrantGeneration = 1
	if err := s.CreateSourceArtifact(ctx, stale); !errors.Is(err, domain.ErrCaptureStaleGrant) {
		t.Fatalf("old generation after pause: %v", err)
	}
	stale.GrantGeneration = 2
	if err := s.CreateSourceArtifact(ctx, stale); !errors.Is(err, domain.ErrCaptureStaleGrant) {
		t.Fatalf("paused generation: %v", err)
	}
	if err := s.CreateSourceArtifact(ctx, captureTestArtifact(now, ax, "ax-still-active")); err != nil {
		t.Fatalf("independent AX grant: %v", err)
	}
	screen.State = domain.CaptureGrantActive
	screen.PausedAt = nil
	screen.PolicyGeneration = 3
	screen.UpdatedAt = now.Add(2 * time.Minute)
	if won, err := s.UpdateCaptureGrantPolicy(ctx, screen, 2); err != nil || !won {
		t.Fatalf("resume policy CAS: won=%v err=%v", won, err)
	}
	stale.GrantGeneration = 3
	if err := s.CreateSourceArtifact(ctx, stale); err != nil {
		t.Fatalf("current generation: %v", err)
	}
	screen.State = domain.CaptureGrantRevoked
	screen.RevokedAt = timePtr(now.Add(3 * time.Minute))
	screen.PolicyGeneration = 4
	screen.UpdatedAt = now.Add(3 * time.Minute)
	if won, err := s.UpdateCaptureGrantPolicy(ctx, screen, 3); err != nil || !won {
		t.Fatalf("revoke CAS: won=%v err=%v", won, err)
	}
	if _, found, err := s.GetSourceArtifact(ctx, screen.OwnerID, stale.ArtifactID); err != nil || found {
		t.Fatalf("revoked grant exposed custody ref: found=%v err=%v", found, err)
	}
	segStart, segEnd := int64(0), int64(10)
	seg := domain.SourceSegment{
		SegmentID: "after-revoke", OwnerID: screen.OwnerID, GrantID: screen.GrantID,
		ArtifactID: stale.ArtifactID, GrantGeneration: 3, Modality: domain.CaptureScreenFrame,
		StartOffset: &segStart, EndOffset: &segEnd, ProvenanceRef: "device/segment",
		LifecycleState: "available", CreatedAt: now,
	}
	if err := s.CreateSourceSegment(ctx, seg); !errors.Is(err, domain.ErrCaptureStaleGrant) {
		t.Fatalf("segment after revoke: %v", err)
	}
	screen.State = domain.CaptureGrantDeleted
	screen.DeletedAt = timePtr(now.Add(4 * time.Minute))
	screen.PolicyGeneration = 5
	screen.UpdatedAt = now.Add(4 * time.Minute)
	if _, err := s.UpdateCaptureGrantPolicy(ctx, screen, 4); !errors.Is(err, domain.ErrCaptureInvalid) {
		t.Fatalf("K4-1 cannot claim cleanup completed: %v", err)
	}
}

func TestCaptureSourceStoreInvalidValues(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	g := captureTestGrant(now, "grant", domain.CaptureScreenFrame)
	g.ProcessingRoute = "provider"
	if err := s.CreateCaptureGrant(ctx, g); !errors.Is(err, domain.ErrCaptureInvalid) {
		t.Fatalf("provider route: %v", err)
	}
	g.ProcessingRoute = "local"
	g.OSPermissionKind = "accessibility"
	if err := s.CreateCaptureGrant(ctx, g); !errors.Is(err, domain.ErrCaptureInvalid) {
		t.Fatalf("wrong OS permission: %v", err)
	}
	g.OSPermissionKind = "screen_recording"
	g.Scope.IncludeWindows = []domain.CaptureTimeWindow{{Start: now, End: now.Add(-time.Second)}}
	if err := s.CreateCaptureGrant(ctx, g); !errors.Is(err, domain.ErrCaptureInvalid) {
		t.Fatalf("invalid time window: %v", err)
	}
	g.Scope.IncludeWindows = nil
	g.Modality = domain.CaptureAudioSegment
	g.OSPermissionKind = "microphone"
	if err := s.CreateCaptureGrant(ctx, g); !errors.Is(err, domain.ErrCaptureInvalid) {
		t.Fatalf("audio not writable in K4-1: %v", err)
	}
}

func TestCaptureSourceStorePeerWriterCannotCommitAfterRevokeAcknowledged(t *testing.T) {
	dir := t.TempDir()
	policyWriter := sqlitetest.MustOpenAt(t, dir)
	sourceWriter := openPeerStore(t, dir)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	grant := captureTestGrant(now, "peer-grant", domain.CaptureScreenFrame)
	if err := policyWriter.CreateCaptureGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	prepared := captureTestArtifact(now, grant, "staged-before-revoke")
	grant.State = domain.CaptureGrantRevoked
	grant.RevokedAt = timePtr(now.Add(time.Minute))
	grant.UpdatedAt = now.Add(time.Minute)
	grant.PolicyGeneration = 2
	if won, err := policyWriter.UpdateCaptureGrantPolicy(ctx, grant, 1); err != nil || !won {
		t.Fatalf("revoke acknowledgement: won=%v err=%v", won, err)
	}
	if err := sourceWriter.CreateSourceArtifact(ctx, prepared); !errors.Is(err, domain.ErrCaptureStaleGrant) {
		t.Fatalf("peer writer committed staged source after revoke: %v", err)
	}
	if _, found, err := sourceWriter.GetSourceArtifact(ctx, prepared.OwnerID, prepared.ArtifactID); err != nil || found {
		t.Fatalf("peer writer exposed stale custody: found=%v err=%v", found, err)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
