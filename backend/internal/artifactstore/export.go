package artifactstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ExportRequest is the owner-triggered delivery boundary. The caller must
// supply the already-read AcceptanceDecision; this package does not infer
// acceptance from provider exit, checks, or a draft label.
type ExportRequest struct {
	Receipt     domain.AttemptReceipt
	Decision    *domain.AcceptanceDecision
	Draft       bool
	Destination string
}

// ExportManifest records the exact retained result and disposition delivered.
type ExportManifest struct {
	AttemptID       string                `json:"attemptId"`
	OutcomeID       string                `json:"outcomeId"`
	ArtifactVersion string                `json:"artifactVersion"`
	Disposition     string                `json:"disposition"`
	ExportedAt      time.Time             `json:"exportedAt"`
	Files           []domain.ArtifactFile `json:"files"`
}

// Export copies an exact retained result to a new destination and writes a
// manifest beside it. Existing non-empty destinations and unsafe paths are
// refused; source artifacts are never removed or mutated.
func (s *Store) Export(ctx context.Context, req ExportRequest) (ExportManifest, error) {
	if err := req.Receipt.Validate(); err != nil {
		return ExportManifest{}, err
	}
	if !req.Receipt.RetentionState.Complete() {
		return ExportManifest{}, errors.New("only a complete retained result can be exported")
	}
	if req.Draft {
		if req.Decision != nil {
			return ExportManifest{}, errors.New("draft export cannot carry an acceptance decision")
		}
	} else if req.Decision == nil || req.Decision.Kind != domain.AcceptanceAccept || req.Decision.ActorType != domain.AcceptanceActorUser {
		return ExportManifest{}, errors.New("accepted export requires the owner's AcceptanceDecision")
	} else if req.Decision.OutcomeID != req.Receipt.OutcomeID {
		return ExportManifest{}, errors.New("acceptance decision belongs to another Outcome")
	}
	if !filepath.IsAbs(req.Destination) {
		return ExportManifest{}, errors.New("export destination must be absolute")
	}
	dest := filepath.Clean(req.Destination)
	if info, err := os.Stat(dest); err == nil {
		if !info.IsDir() {
			return ExportManifest{}, errors.New("export destination is not a directory")
		}
		entries, readErr := os.ReadDir(dest)
		if readErr != nil {
			return ExportManifest{}, readErr
		}
		if len(entries) != 0 {
			return ExportManifest{}, errors.New("export destination is not empty")
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dest, 0o750); err != nil {
			return ExportManifest{}, err
		}
	} else {
		return ExportManifest{}, err
	}
	for _, file := range req.Receipt.Files {
		if file.ChangeKind == domain.ArtifactDeleted {
			continue
		}
		body, mode, err := s.Read(ctx, req.Receipt, file)
		if err != nil {
			return ExportManifest{}, fmt.Errorf("read export input %s: %w", file.RelativePath, err)
		}
		full, err := confinedPath(dest, file.RelativePath)
		if err != nil {
			return ExportManifest{}, err
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return ExportManifest{}, err
		}
		if err := os.WriteFile(full, body, mode.Perm()); err != nil {
			return ExportManifest{}, err
		}
	}
	manifest := ExportManifest{AttemptID: string(req.Receipt.AttemptID), OutcomeID: string(req.Receipt.OutcomeID), ArtifactVersion: req.Receipt.ArtifactVersion, Disposition: "accepted", ExportedAt: time.Now().UTC(), Files: append([]domain.ArtifactFile(nil), req.Receipt.Files...)}
	if req.Draft {
		manifest.Disposition = "draft"
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return ExportManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(dest, "KENNEL-EXPORT.json"), append(encoded, '\n'), 0o600); err != nil {
		return ExportManifest{}, err
	}
	return manifest, nil
}
