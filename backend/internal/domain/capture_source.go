package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// CaptureModality identifies a separately governed source kind. Capture
// records are untrusted metadata, never memory or responsibility authority.
type CaptureModality string

// K4 modality vocabulary; audio is reserved for a later slice.
const (
	CaptureScreenFrame  CaptureModality = "screen_frame"
	CaptureAXTree       CaptureModality = "ax_tree"
	CaptureAudioSegment CaptureModality = "audio_segment"
)

// Valid reports whether a modality belongs to the persisted vocabulary.
func (m CaptureModality) Valid() bool {
	return m == CaptureScreenFrame || m == CaptureAXTree || m == CaptureAudioSegment
}

// K4Writable reports whether the K4-1 store accepts this modality.
func (m CaptureModality) K4Writable() bool { return m == CaptureScreenFrame || m == CaptureAXTree }

// CaptureGrantState is the visible consent lifecycle state.
type CaptureGrantState string

// Grant states are independent of OS permission state.
const (
	CaptureGrantActive  CaptureGrantState = "active"
	CaptureGrantPaused  CaptureGrantState = "paused"
	CaptureGrantDenied  CaptureGrantState = "denied"
	CaptureGrantStale   CaptureGrantState = "stale"
	CaptureGrantFailed  CaptureGrantState = "failed"
	CaptureGrantRevoked CaptureGrantState = "revoked"
	CaptureGrantDeleted CaptureGrantState = "deleted"
)

// Valid reports whether the state belongs to the persisted vocabulary.
func (s CaptureGrantState) Valid() bool {
	switch s {
	case CaptureGrantActive, CaptureGrantPaused, CaptureGrantDenied, CaptureGrantStale, CaptureGrantFailed, CaptureGrantRevoked, CaptureGrantDeleted:
		return true
	}
	return false
}

// CaptureTimeWindow is a bounded inclusion or exclusion interval.
type CaptureTimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CaptureScope is bounded JSON so each modality can carry evolving inclusion
// and exclusion rules without a global permission bit. Empty inclusion means
// no extra app/person/space filter; exclusions always take precedence.
type CaptureScope struct {
	IncludeApps    []string            `json:"include_apps"`
	ExcludeApps    []string            `json:"exclude_apps"`
	IncludePeople  []string            `json:"include_people"`
	ExcludePeople  []string            `json:"exclude_people"`
	IncludeSpaces  []string            `json:"include_spaces"`
	ExcludeSpaces  []string            `json:"exclude_spaces"`
	IncludeWindows []CaptureTimeWindow `json:"include_windows"`
	ExcludeWindows []CaptureTimeWindow `json:"exclude_windows"`
}

// Validate rejects oversized, duplicate, or malformed scope entries.
func (s CaptureScope) Validate() error {
	for _, list := range [][]string{s.IncludeApps, s.ExcludeApps, s.IncludePeople, s.ExcludePeople, s.IncludeSpaces, s.ExcludeSpaces} {
		if len(list) > 128 {
			return ErrCaptureInvalid
		}
		seen := map[string]bool{}
		for _, v := range list {
			if strings.TrimSpace(v) != v || v == "" || len(v) > 256 || seen[v] {
				return ErrCaptureInvalid
			}
			seen[v] = true
		}
	}
	for _, list := range [][]CaptureTimeWindow{s.IncludeWindows, s.ExcludeWindows} {
		if len(list) > 64 {
			return ErrCaptureInvalid
		}
		for _, w := range list {
			if w.Start.IsZero() || !w.End.After(w.Start) {
				return ErrCaptureInvalid
			}
		}
	}
	encoded, err := json.Marshal(s)
	if err != nil || len(encoded) > 32768 {
		return ErrCaptureInvalid
	}
	return nil
}

// CaptureGrant is the owner-scoped policy for one source modality.
type CaptureGrant struct {
	GrantID                 string
	OwnerID                 string
	DeviceID                string
	Modality                CaptureModality
	OSPermissionKind        string
	OSPermissionState       string
	Scope                   CaptureScope
	Purpose                 string
	AllowedSpaceIDs         []string
	ProcessingRoute         string
	DisclosurePolicy        string
	RawRetentionSeconds     int64
	DerivedRetentionSeconds int64
	Sensitivity             string
	BystanderPolicy         string
	ExportBehavior          string
	DeleteBehavior          string
	State                   CaptureGrantState
	PolicyGeneration        int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
	PausedAt                *time.Time
	RevokedAt               *time.Time
	DeletedAt               *time.Time
}

// Validate checks the grant vocabulary, bounds, timestamps, and permissions.
func (g CaptureGrant) Validate() error {
	if !validCaptureID(g.GrantID) || !validCaptureID(g.OwnerID) || !validCaptureID(g.DeviceID) || !g.Modality.Valid() || !g.State.Valid() || g.PolicyGeneration < 1 || g.CreatedAt.IsZero() || g.UpdatedAt.Before(g.CreatedAt) || g.Scope.Validate() != nil {
		return ErrCaptureInvalid
	}
	kind := map[CaptureModality]string{CaptureScreenFrame: "screen_recording", CaptureAXTree: "accessibility", CaptureAudioSegment: "microphone"}[g.Modality]
	if g.OSPermissionKind != kind || !oneOf(g.OSPermissionState, "unknown", "not_requested", "granted", "denied", "restricted") || (g.State == CaptureGrantActive && g.OSPermissionState != "granted") {
		return ErrCaptureInvalid
	}
	if strings.TrimSpace(g.Purpose) == "" || len(g.Purpose) > 1024 || len(g.AllowedSpaceIDs) > 64 || g.ProcessingRoute != "local" || !oneOf(g.DisclosurePolicy, "local_only", "owner_review") || g.RawRetentionSeconds < 0 || g.RawRetentionSeconds > 2592000 || g.DerivedRetentionSeconds < 0 || g.DerivedRetentionSeconds > 31536000 || !oneOf(g.Sensitivity, "ordinary", "sensitive", "restricted") || !oneOf(g.BystanderPolicy, "exclude", "redact_before_derive", "owner_review") || !oneOf(g.ExportBehavior, "owner_request_only", "prohibited") || !oneOf(g.DeleteBehavior, "cascade_local_custody", "retain_content_free_receipt") {
		return ErrCaptureInvalid
	}
	seen := map[string]bool{}
	for _, id := range g.AllowedSpaceIDs {
		if !validCaptureID(id) || len(id) > 256 || seen[id] {
			return ErrCaptureInvalid
		}
		seen[id] = true
	}
	spaces, err := json.Marshal(g.AllowedSpaceIDs)
	if err != nil || len(spaces) > 8192 {
		return ErrCaptureInvalid
	}
	if g.State == CaptureGrantRevoked && g.RevokedAt == nil || g.State == CaptureGrantDeleted && g.DeletedAt == nil {
		return ErrCaptureInvalid
	}
	if g.State == CaptureGrantPaused && g.PausedAt == nil || g.State != CaptureGrantDeleted && g.DeletedAt != nil || g.State != CaptureGrantRevoked && g.State != CaptureGrantDeleted && g.RevokedAt != nil {
		return ErrCaptureInvalid
	}
	for _, at := range []*time.Time{g.PausedAt, g.RevokedAt, g.DeletedAt} {
		if at != nil && (at.Before(g.CreatedAt) || at.After(g.UpdatedAt)) {
			return ErrCaptureInvalid
		}
	}
	return nil
}

// SourceArtifact names one untrusted locally custodied source.
type SourceArtifact struct {
	ArtifactID      string
	OwnerID         string
	GrantID         string
	GrantGeneration int64
	Modality        CaptureModality
	ContentKind     string
	MIMEType        string
	HashAlgorithm   string
	ContentHash     string
	BlobCustodyRef  string
	ProvenanceKind  string
	ProvenanceRef   string
	CapturedAt      time.Time
	LifecycleState  string
	CreatedAt       time.Time
	DeletedAt       *time.Time
}

// Validate checks content address, custody metadata, and lineage fields.
func (a SourceArtifact) Validate() error {
	if !validCaptureID(a.ArtifactID) || !validCaptureID(a.OwnerID) || !validCaptureID(a.GrantID) || a.GrantGeneration < 1 || !a.Modality.Valid() || strings.TrimSpace(a.MIMEType) == "" || len(a.MIMEType) > 255 || a.HashAlgorithm != "sha256" || len(a.ContentHash) != 64 || !strings.HasPrefix(a.BlobCustodyRef, "local:") || len(a.BlobCustodyRef) <= 6 || !validCaptureID(a.ProvenanceRef) || len(a.ProvenanceRef) > 1024 || a.CapturedAt.IsZero() || a.CreatedAt.IsZero() || !oneOf(a.LifecycleState, "available", "deletion_pending", "deleted") || (a.LifecycleState == "deleted" && a.DeletedAt == nil) {
		return ErrCaptureInvalid
	}
	for _, c := range a.ContentHash {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return ErrCaptureInvalid
		}
	}
	if a.Modality == CaptureScreenFrame && (a.ContentKind != "image" || a.ProvenanceKind != "screen_capture") || a.Modality == CaptureAXTree && (a.ContentKind != "structured" || a.ProvenanceKind != "accessibility_snapshot") || a.Modality == CaptureAudioSegment && (a.ContentKind != "audio" || a.ProvenanceKind != "audio_capture") {
		return ErrCaptureInvalid
	}
	if a.Modality == CaptureScreenFrame && !strings.HasPrefix(a.MIMEType, "image/") ||
		a.Modality == CaptureAXTree && a.MIMEType != "application/json" && a.MIMEType != "application/vnd.kennel.ax+json" ||
		a.Modality == CaptureAudioSegment && !strings.HasPrefix(a.MIMEType, "audio/") {
		return ErrCaptureInvalid
	}
	return nil
}

// SourceSegment locates a bounded interval within one artifact.
type SourceSegment struct {
	SegmentID       string
	OwnerID         string
	GrantID         string
	ArtifactID      string
	GrantGeneration int64
	Modality        CaptureModality
	Ordinal         int64
	StartOffset     *int64
	EndOffset       *int64
	StartAt         *time.Time
	EndAt           *time.Time
	ProvenanceRef   string
	LifecycleState  string
	CreatedAt       time.Time
	DeletedAt       *time.Time
}

// Validate checks the segment interval, ordering, and source lineage fields.
func (s SourceSegment) Validate() error {
	if !validCaptureID(s.SegmentID) || !validCaptureID(s.OwnerID) || !validCaptureID(s.GrantID) || !validCaptureID(s.ArtifactID) || s.GrantGeneration < 1 || !s.Modality.Valid() || s.Ordinal < 0 || !validCaptureID(s.ProvenanceRef) || len(s.ProvenanceRef) > 1024 || s.CreatedAt.IsZero() || !oneOf(s.LifecycleState, "available", "deletion_pending", "deleted") || (s.LifecycleState == "deleted" && s.DeletedAt == nil) {
		return ErrCaptureInvalid
	}
	byOffset := s.StartOffset != nil && s.EndOffset != nil && *s.StartOffset >= 0 && *s.EndOffset > *s.StartOffset && s.StartAt == nil && s.EndAt == nil
	byTime := s.StartAt != nil && s.EndAt != nil && !s.StartAt.IsZero() && !s.EndAt.Before(*s.StartAt) && s.StartOffset == nil && s.EndOffset == nil
	if !byOffset && !byTime {
		return ErrCaptureInvalid
	}
	return nil
}

func validCaptureID(s string) bool { return s != "" && strings.TrimSpace(s) == s && len(s) <= 256 }
func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// ErrCaptureInvalid reports malformed or unsupported K4-1 source metadata.
var ErrCaptureInvalid = errors.New("invalid capture source record")

// ErrCaptureConflict reports a duplicate stable identity or content address.
var ErrCaptureConflict = errors.New("capture source conflict")

// ErrCaptureStaleGrant reports a grant that changed before source persistence.
var ErrCaptureStaleGrant = errors.New("capture grant generation or state is stale")
