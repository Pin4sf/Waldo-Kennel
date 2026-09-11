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

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// ExportVerification is what reading a destination establishes about a delivery
// whose ledger row never reached a terminal state.
//
// Only Verified permits recording the transfer as successful. The other three
// are deliberately distinct rather than collapsed into "failed": they lead the
// owner somewhere different, and reporting "nothing was delivered" when in fact
// something unrecognised is sitting at the destination would be worse than
// saying so.
type ExportVerification string

const (
	// ExportVerified means the destination holds exactly this authorized
	// delivery: the manifest names this Attempt, artifact version, disposition
	// and acceptance, and every retained file is present with the digest the
	// retained result recorded. It proves the transfer completed. It says
	// nothing about Outcome acceptance.
	ExportVerified ExportVerification = "verified"
	// ExportAbsent means there is nothing at the destination, so the transfer
	// did not happen.
	ExportAbsent ExportVerification = "absent"
	// ExportMismatch means something is at the destination and it is not this
	// delivery. Nothing may be concluded about the transfer, and nothing may be
	// written over what is there.
	ExportMismatch ExportVerification = "mismatch"
	// ExportUnreadable means the destination could not be read well enough to
	// decide. Not deciding is different from deciding against, and is reported
	// as itself.
	ExportUnreadable ExportVerification = "unreadable"
)

// Conclusive reports whether this verification establishes a completed
// transfer. Only an exact match does.
func (v ExportVerification) Conclusive() bool { return v == ExportVerified }

// VerifyExportRequest is the authorized delivery a destination is checked
// against. It is the same identity the export itself was bound to, so a
// recovered success can never describe a different delivery than the one the
// owner requested.
type VerifyExportRequest struct {
	Destination string
	// Receipt is the retained result that was authorized for transfer. Its
	// file list and digests are the authority for what must be present.
	Receipt              domain.AttemptReceipt
	ContractRevisionID   domain.ContractRevisionID
	AcceptanceDecisionID domain.AcceptanceDecisionID
	Draft                bool
}

// ExportVerificationResult is one read-only verdict on a destination.
type ExportVerificationResult struct {
	Verification ExportVerification
	// Detail is the stable human-readable reason, carried into the delivery
	// ledger so an owner can act on it without re-inspecting the filesystem.
	Detail       string
	ManifestPath string
	FileCount    int
	ByteCount    int64
}

// VerifyExport reads a destination and reports whether it holds exactly one
// authorized delivery.
//
// It is strictly read-only. Nothing is created, replaced, removed or retried:
// its whole purpose is to let a lost ledger row be resolved from evidence that
// already exists, and a recovery that wrote anything would be a second
// transfer rather than a reading of the first.
//
// Every artifact is read with no symbolic link anywhere between the destination
// and the file, so nothing outside the delivered tree can count as delivered
// content — see readDeliveredFile, which is where that is enforced and why a
// leaf check alone is not enough.
//
// A symlinked *parent of the destination* is followed, exactly as the export
// followed it. That is DLV-02 and it is deliberate here: the export wrote
// through that link, so the bytes really are behind it and refusing to look
// would report a completed transfer as missing. The open half of DLV-02 is that
// neither the ledger nor this result records the resolved path, so both still
// name where the delivery was asked to go rather than where it is.
func (s *Store) VerifyExport(ctx context.Context, req VerifyExportRequest) ExportVerificationResult {
	if err := req.Receipt.Validate(); err != nil {
		return ExportVerificationResult{Verification: ExportUnreadable,
			Detail: "the authorized retained result is invalid: " + err.Error()}
	}
	if !filepath.IsAbs(req.Destination) {
		return ExportVerificationResult{Verification: ExportUnreadable,
			Detail: "destination must be absolute"}
	}
	dest := filepath.Clean(req.Destination)

	info, err := os.Lstat(dest)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return ExportVerificationResult{Verification: ExportAbsent,
			Detail: "nothing exists at the destination, so no transfer completed"}
	case err != nil:
		return ExportVerificationResult{Verification: ExportUnreadable,
			Detail: "the destination could not be read: " + err.Error()}
	case info.Mode()&os.ModeSymlink != 0:
		// The export refuses a symlink destination outright, so one here was
		// not produced by this delivery.
		return ExportVerificationResult{Verification: ExportMismatch,
			Detail: "the destination is a symlink, which this delivery could not have created"}
	case !info.IsDir():
		return ExportVerificationResult{Verification: ExportMismatch,
			Detail: "the destination is not a delivery directory"}
	}

	manifestPath := filepath.Join(dest, manifestName)
	encoded, err := os.ReadFile(manifestPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// Something is there, but it is not one of our deliveries. Saying
		// "absent" would invite a retry that overwrites whatever it is.
		return ExportVerificationResult{Verification: ExportMismatch,
			Detail: "the destination holds content with no delivery manifest"}
	case err != nil:
		return ExportVerificationResult{Verification: ExportUnreadable,
			Detail: "the delivery manifest could not be read: " + err.Error()}
	}
	var manifest ExportManifest
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return ExportVerificationResult{Verification: ExportUnreadable,
			Detail: "the delivery manifest could not be parsed: " + err.Error()}
	}

	if mismatch := manifestIdentityMismatch(manifest, req); mismatch != "" {
		return ExportVerificationResult{Verification: ExportMismatch, Detail: mismatch, ManifestPath: manifestPath}
	}
	result := ExportVerificationResult{ManifestPath: manifestPath}
	if mismatch := manifestFileSetMismatch(manifest, req.Receipt); mismatch != "" {
		result.Verification, result.Detail = ExportMismatch, mismatch
		return result
	}

	// The manifest agrees with the authorization; now the bytes have to agree
	// with the manifest. A manifest alone would only prove that an export
	// started here.
	for _, file := range req.Receipt.Files {
		if file.ChangeKind == domain.ArtifactDeleted {
			continue
		}
		full, err := confinedPath(dest, file.RelativePath)
		if err != nil {
			result.Verification = ExportMismatch
			result.Detail = fmt.Sprintf("delivered path %q escapes the destination", file.RelativePath)
			return result
		}
		body, err := readDeliveredFile(ctx, dest, full)
		switch {
		case errors.Is(err, os.ErrNotExist):
			result.Verification = ExportMismatch
			result.Detail = fmt.Sprintf("delivered artifact %q is missing from the destination", file.RelativePath)
			return result
		case errors.Is(err, errDeliveredNotRegular):
			result.Verification = ExportMismatch
			result.Detail = fmt.Sprintf("delivered path %q is not a regular file", file.RelativePath)
			return result
		case errors.Is(err, errDeliveredLinkedComponent):
			result.Verification = ExportMismatch
			result.Detail = fmt.Sprintf("delivered path %q is reached through a symbolic link, so it is not content this delivery wrote", file.RelativePath)
			return result
		case err != nil:
			result.Verification = ExportUnreadable
			result.Detail = fmt.Sprintf("delivered artifact %q could not be read: %v", file.RelativePath, err)
			return result
		}
		digest := sha256.Sum256(body)
		if hex.EncodeToString(digest[:]) != file.ContentDigest {
			result.Verification = ExportMismatch
			result.Detail = fmt.Sprintf("delivered artifact %q does not match the retained digest", file.RelativePath)
			return result
		}
		result.FileCount++
		if file.SizeBytes != nil {
			result.ByteCount += *file.SizeBytes
		}
	}
	// FileCount counts every retained file, deleted ones included, to match
	// what a live delivery records.
	result.FileCount = len(req.Receipt.Files)
	result.Verification = ExportVerified
	result.Detail = "the destination holds exactly this authorized delivery; transfer completion was established by reading it, not by observing it"
	return result
}

var (
	// errDeliveredNotRegular means a delivered path exists but is a symlink,
	// directory or device rather than the file that was retained.
	errDeliveredNotRegular = errors.New("delivered path is not a regular file")
	// errDeliveredLinkedComponent means some directory between the destination
	// and a delivered artifact is a symbolic link.
	errDeliveredLinkedComponent = errors.New("delivered path is reached through a symbolic link")
)

// readDeliveredFile reads one delivered artifact from inside the destination,
// refusing any symbolic link on the way to it.
//
// Checking only the final component is not enough, and this is the whole point
// of the function. confinedPath is lexical, so a *directory* between the
// destination and the artifact escapes it: replace `dest/nested` with a link to
// another directory and `dest/nested/deep.txt` still joins to a path under
// `dest`, while the file found through it is an ordinary regular file
// somewhere else entirely. Since a digest matches identical bytes wherever they
// live, verification would then confirm a delivery whose artifacts were never
// at the destination.
//
// Every component below the destination is therefore checked, not just the
// leaf. The export creates only real directories and regular files inside the
// bundle it commits, so a link anywhere in that tree is not content this
// delivery wrote. Following a symlinked parent *of* the destination stays
// deliberate and is resolved by the caller before this is ever reached: the
// export wrote through it, so reading through it is how the bytes are found.
func readDeliveredFile(ctx context.Context, root, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	// Walk down from the destination so the first link encountered is reported,
	// rather than whatever the kernel resolved the whole path to.
	walked := root
	segments := strings.Split(relative, string(filepath.Separator))
	for i, segment := range segments {
		if segment == "" || segment == "." {
			continue
		}
		walked = filepath.Join(walked, segment)
		info, err := os.Lstat(walked)
		if err != nil {
			return nil, err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return nil, errDeliveredLinkedComponent
		case i == len(segments)-1:
			if !info.Mode().IsRegular() {
				return nil, errDeliveredNotRegular
			}
		case !info.IsDir():
			// A non-directory where a directory has to be: whatever this
			// delivery wrote, it is not still here.
			return nil, errDeliveredNotRegular
		}
	}
	return os.ReadFile(path)
}

// manifestIdentityMismatch reports the first way the manifest describes a
// different delivery than the one authorized.
func manifestIdentityMismatch(manifest ExportManifest, req VerifyExportRequest) string {
	disposition := "accepted"
	if req.Draft {
		disposition = "draft"
	}
	switch {
	case manifest.AttemptID != string(req.Receipt.AttemptID):
		return fmt.Sprintf("the destination holds Attempt %q, not %q", manifest.AttemptID, req.Receipt.AttemptID)
	case manifest.OutcomeID != string(req.Receipt.OutcomeID):
		return fmt.Sprintf("the destination holds Outcome %q, not %q", manifest.OutcomeID, req.Receipt.OutcomeID)
	case manifest.ArtifactVersion != req.Receipt.ArtifactVersion:
		return fmt.Sprintf("the destination holds artifact version %q, not %q", manifest.ArtifactVersion, req.Receipt.ArtifactVersion)
	case manifest.Disposition != disposition:
		return fmt.Sprintf("the destination holds a %q delivery, not %q", manifest.Disposition, disposition)
	case manifest.ContractRevisionID != string(req.ContractRevisionID):
		return fmt.Sprintf("the destination names Contract revision %q, not %q", manifest.ContractRevisionID, req.ContractRevisionID)
	case manifest.AcceptanceDecisionID != string(req.AcceptanceDecisionID):
		// An accepted delivery that names a different decision was authorized
		// by a review this one did not have.
		return fmt.Sprintf("the destination names AcceptanceDecision %q, not %q", manifest.AcceptanceDecisionID, req.AcceptanceDecisionID)
	case manifest.ContractRevisionNumber != req.Receipt.ContractRevisionNumber:
		return fmt.Sprintf("the destination names Contract revision number %d, not %d", manifest.ContractRevisionNumber, req.Receipt.ContractRevisionNumber)
	}
	return ""
}

// manifestFileSetMismatch reports a manifest that does not describe the exact
// retained file set, which means the destination holds a different export of
// the same identity.
func manifestFileSetMismatch(manifest ExportManifest, receipt domain.AttemptReceipt) string {
	if len(manifest.Files) != len(receipt.Files) {
		return fmt.Sprintf("the destination manifest lists %d files; the retained result has %d",
			len(manifest.Files), len(receipt.Files))
	}
	recorded := make(map[string]domain.ArtifactFile, len(manifest.Files))
	for _, file := range manifest.Files {
		recorded[file.RelativePath] = file
	}
	for _, file := range receipt.Files {
		found, ok := recorded[file.RelativePath]
		if !ok {
			return fmt.Sprintf("the destination manifest does not list %q", file.RelativePath)
		}
		if found.ContentDigest != file.ContentDigest {
			return fmt.Sprintf("the destination manifest records a different digest for %q", file.RelativePath)
		}
		if found.ChangeKind != file.ChangeKind {
			return fmt.Sprintf("the destination manifest records %q as %q, not %q", file.RelativePath, found.ChangeKind, file.ChangeKind)
		}
	}
	return ""
}
