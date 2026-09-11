package artifactstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

var (
	// ErrExportDestinationConflict means the owner-selected destination already exists.
	ErrExportDestinationConflict = errors.New("export destination conflicts with existing content")
	// ErrExportDestinationUnsafe means the owner-selected destination cannot be safely confined.
	ErrExportDestinationUnsafe = errors.New("export destination is unsafe")
	// ErrExportManifestCollision means retained content uses the reserved manifest name.
	ErrExportManifestCollision = errors.New("export manifest name collides with retained content")
	// ErrExportArtifactMissing means a retained blob could not be verified during export.
	ErrExportArtifactMissing = errors.New("retained export artifact is missing or corrupt")
)

// ExportRequest is the owner-triggered delivery boundary. The caller must
// supply the already-read AcceptanceDecision; this package does not infer
// acceptance from provider exit, checks, or a draft label.
//
// An accepted export must also name what was accepted. A decision carries an
// Outcome and a Contract revision, never an artifact version, so the store
// cannot resolve on its own whether the owner reviewed *this* result. The
// caller resolves that once and states it here; the store then refuses any
// mismatch rather than assuming the newest retained bytes are the reviewed
// ones.
type ExportRequest struct {
	Receipt  domain.AttemptReceipt
	Decision *domain.AcceptanceDecision
	// ContractRevisionID is the revision the receipt itself belongs to,
	// resolved by the caller from the receipt's Outcome and revision number.
	ContractRevisionID domain.ContractRevisionID
	// AcceptedArtifactVersion is the artifact version the owner reviewed. It
	// must equal the receipt's version for an accepted export.
	AcceptedArtifactVersion string
	Draft                   bool
	Destination             string
}

// ManifestName is reserved for delivery metadata. A retained artifact may
// legitimately use it, so a collision is refused rather than resolved.
const ManifestName = "KENNEL-EXPORT.json"

const manifestName = ManifestName

// ExportManifest records the exact retained result and disposition delivered.
// Deleted paths stay listed: a delivery that silently drops them would not
// describe what the Attempt actually did.
type ExportManifest struct {
	AttemptID              string                `json:"attemptId"`
	OutcomeID              string                `json:"outcomeId"`
	ArtifactVersion        string                `json:"artifactVersion"`
	Disposition            string                `json:"disposition"`
	ExportedAt             time.Time             `json:"exportedAt"`
	ContractRevisionNumber int64                 `json:"contractRevisionNumber"`
	ContractRevisionID     string                `json:"contractRevisionId,omitempty"`
	AcceptanceDecisionID   string                `json:"acceptanceDecisionId,omitempty"`
	WorkspaceKind          string                `json:"workspaceKind"`
	RepositoryIdentity     string                `json:"repositoryIdentity,omitempty"`
	BaseRevision           string                `json:"baseRevision,omitempty"`
	ResultRevision         string                `json:"resultRevision,omitempty"`
	Files                  []domain.ArtifactFile `json:"files"`
}

// bindAcceptance refuses any accepted export that cannot prove the owner
// reviewed this exact artifact under this exact Contract revision.
func bindAcceptance(req ExportRequest) error {
	if req.Decision == nil {
		return errors.New("accepted export requires the owner's AcceptanceDecision")
	}
	if err := req.Decision.Validate(); err != nil {
		return fmt.Errorf("acceptance decision is invalid: %w", err)
	}
	if req.Decision.Kind != domain.AcceptanceAccept || req.Decision.ActorType != domain.AcceptanceActorUser {
		return errors.New("accepted export requires the owner's AcceptanceDecision")
	}
	if req.Decision.OutcomeID != req.Receipt.OutcomeID {
		return errors.New("acceptance decision belongs to another Outcome")
	}
	if req.ContractRevisionID.IsZero() {
		return errors.New("accepted export requires the receipt's Contract revision identity")
	}
	if req.Decision.ContractRevisionID != req.ContractRevisionID {
		// Same Outcome, older review. Accepting it here would label an
		// unreviewed revision as accepted.
		return errors.New("acceptance decision belongs to another Contract revision")
	}
	if strings.TrimSpace(req.AcceptedArtifactVersion) == "" {
		return errors.New("accepted export requires the reviewed artifact version")
	}
	if req.AcceptedArtifactVersion != req.Receipt.ArtifactVersion {
		return errors.New("acceptance names a different artifact version than the retained result")
	}
	return nil
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
	} else if err := bindAcceptance(req); err != nil {
		return ExportManifest{}, err
	}
	// Refuse a reserved-name collision before anything is written. Writing the
	// payload and then overwriting it with metadata would report success while
	// delivering a different file than the one that was retained.
	for _, file := range req.Receipt.Files {
		if file.ChangeKind == domain.ArtifactDeleted {
			continue
		}
		if strings.EqualFold(file.RelativePath, manifestName) {
			return ExportManifest{}, fmt.Errorf("%w: %q", ErrExportManifestCollision, file.RelativePath)
		}
	}
	if !filepath.IsAbs(req.Destination) {
		return ExportManifest{}, errors.New("export destination must be absolute")
	}
	dest := filepath.Clean(req.Destination)
	if info, err := os.Lstat(dest); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return ExportManifest{}, fmt.Errorf("%w: destination is a symlink", ErrExportDestinationUnsafe)
		}
		return ExportManifest{}, fmt.Errorf("%w: destination already exists", ErrExportDestinationConflict)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ExportManifest{}, fmt.Errorf("%w: %w", ErrExportDestinationUnsafe, err)
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return ExportManifest{}, err
	}
	stage := filepath.Join(parent, "."+filepath.Base(dest)+".kennel-export-"+uuid.NewString())
	if err := os.Mkdir(stage, 0o750); err != nil {
		return ExportManifest{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stage)
		}
	}()
	for _, file := range req.Receipt.Files {
		if file.ChangeKind == domain.ArtifactDeleted {
			continue
		}
		body, mode, err := s.Read(ctx, req.Receipt, file)
		if err != nil {
			return ExportManifest{}, fmt.Errorf("%w: read %s: %w", ErrExportArtifactMissing, file.RelativePath, err)
		}
		full, err := confinedPath(stage, file.RelativePath)
		if err != nil {
			return ExportManifest{}, fmt.Errorf("%w: %w", ErrExportDestinationUnsafe, err)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return ExportManifest{}, err
		}
		if err := os.WriteFile(full, body, mode.Perm()); err != nil {
			return ExportManifest{}, err
		}
		// Verify what actually landed against the retained manifest entry, so a
		// short or intercepted write cannot be delivered as the accepted result.
		written, err := os.ReadFile(full)
		if err != nil {
			return ExportManifest{}, err
		}
		digest := sha256.Sum256(written)
		if hex.EncodeToString(digest[:]) != file.ContentDigest {
			return ExportManifest{}, fmt.Errorf("exported %s does not match the retained digest", file.RelativePath)
		}
	}
	manifest := ExportManifest{
		AttemptID: string(req.Receipt.AttemptID), OutcomeID: string(req.Receipt.OutcomeID),
		ArtifactVersion: req.Receipt.ArtifactVersion, Disposition: "accepted", ExportedAt: time.Now().UTC(),
		ContractRevisionNumber: req.Receipt.ContractRevisionNumber, ContractRevisionID: string(req.ContractRevisionID),
		WorkspaceKind: string(req.Receipt.WorkspaceKind), RepositoryIdentity: req.Receipt.RepositoryIdentity,
		BaseRevision: req.Receipt.BaseRevision, ResultRevision: req.Receipt.ResultRevision,
		Files: append([]domain.ArtifactFile(nil), req.Receipt.Files...),
	}
	if req.Decision != nil {
		manifest.AcceptanceDecisionID = string(req.Decision.ID)
	}
	if req.Draft {
		manifest.Disposition = "draft"
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return ExportManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(stage, manifestName), append(encoded, '\n'), 0o600); err != nil {
		return ExportManifest{}, err
	}
	if err := os.Rename(stage, dest); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ExportManifest{}, fmt.Errorf("%w: destination appeared during export", ErrExportDestinationConflict)
		}
		return ExportManifest{}, err
	}
	committed = true
	return manifest, nil
}
