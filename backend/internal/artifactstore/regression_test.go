package artifactstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func stagedInput(attempt string, workspace string) Input {
	return Input{
		AttemptID: domain.AttemptID(attempt), OutcomeID: "outcome-1", PlanRevisionID: "plan-1",
		WorkUnitID: "unit-1", ContractRevisionNumber: 1,
		WorkspaceKind: domain.WorkspaceStagedFolder, WorkspacePath: workspace,
	}
}

func ownerDecision(contract domain.ContractRevisionID) *domain.AcceptanceDecision {
	return &domain.AcceptanceDecision{
		ID: "accept-1", OutcomeID: "outcome-1", ContractRevisionID: contract,
		Kind: domain.AcceptanceAccept, ActorType: domain.AcceptanceActorUser, Summary: "Reviewed",
		ResourceDisposition: domain.ResourceDispositionRetain, RequestKey: "request-1",
		RequestFingerprint: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}

// ST1. A file the byte bound declines is still part of what the Attempt
// produced. The receipt must persist, report incomplete, and name the path it
// could not retain -- not fail validation and lose the whole snapshot.
func TestRetainKeepsAValidReceiptWhenTheByteBoundDeclinesAFile(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "large.txt"), []byte("12345"), 0o640); err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts"), MaxBytes: 4})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), stagedInput("bounded-attempt", workspace))
	if err != nil {
		t.Fatalf("bounded retention returned an error instead of an incomplete receipt: %v", err)
	}
	if err := result.Receipt.Validate(); err != nil {
		t.Fatalf("bounded receipt is not persistable: %v", err)
	}
	if result.Receipt.RetentionState != domain.RetentionIncomplete {
		t.Fatalf("retention state = %s, want incomplete", result.Receipt.RetentionState)
	}
	if result.Receipt.RetentionState.Complete() {
		t.Fatal("an incomplete snapshot must never satisfy a handoff")
	}
	if len(result.Receipt.Files) != 1 {
		t.Fatalf("files = %d, want the declined path recorded", len(result.Receipt.Files))
	}
	file := result.Receipt.Files[0]
	if file.ContentDigest != "" {
		t.Fatal("declined content must not carry a digest")
	}
	if !strings.Contains(file.UnsupportedReason, "content declined") {
		t.Fatalf("unsupported reason = %q, want the specific decline", file.UnsupportedReason)
	}
	if result.PublishedNew {
		t.Fatal("an incomplete snapshot must not publish a content version")
	}
}

// A bound reached partway through must not be reported as a clean snapshot,
// and the files that did fit stay retained with their digests.
func TestRetainReportsIncompleteAfterPartialCapture(t *testing.T) {
	workspace := t.TempDir()
	for name, body := range map[string]string{"a.txt": "aaaa", "b.txt": "bbbb"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte(body), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts"), MaxBytes: 6})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), stagedInput("partial-attempt", workspace))
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.RetentionState != domain.RetentionIncomplete {
		t.Fatalf("state = %s, want incomplete", result.Receipt.RetentionState)
	}
	var retained, declined int
	for _, file := range result.Receipt.Files {
		if file.ContentDigest != "" {
			retained++
		} else {
			declined++
		}
	}
	if retained != 1 || declined != 1 {
		t.Fatalf("retained/declined = %d/%d, want 1/1", retained, declined)
	}
}

// ST3. A retained artifact may legitimately be named KENNEL-EXPORT.json.
// Exporting it must refuse the collision before writing anything, rather than
// replacing the payload with metadata and reporting success.
func TestExportRefusesReservedManifestNameCollision(t *testing.T) {
	workspace := t.TempDir()
	payload := []byte(`{"produced":"by the attempt"}`)
	if err := os.WriteFile(filepath.Join(workspace, manifestName), payload, 0o640); err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), stagedInput("collision-attempt", workspace))
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "export")
	if _, err := store.Export(context.Background(), ExportRequest{Receipt: result.Receipt, Draft: true, Destination: destination}); err == nil {
		t.Fatal("export overwrote a retained artifact with its own manifest")
	}
	if entries, err := os.ReadDir(destination); err == nil && len(entries) != 0 {
		t.Fatalf("refused export left %d entries behind", len(entries))
	}
}

// SP3. Acceptance names an Outcome and a Contract revision, never an artifact.
// An older decision for the same Outcome must not label a different revision
// or a different result as accepted.
func TestExportRefusesStaleOrMismatchedAcceptance(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "report.md"), []byte("v2"), 0o640); err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), stagedInput("binding-attempt", workspace))
	if err != nil {
		t.Fatal(err)
	}
	version := result.Receipt.ArtifactVersion
	for _, tc := range []struct {
		name string
		req  ExportRequest
	}{
		{"acceptance from an older Contract revision", ExportRequest{Decision: ownerDecision("contract-1"), ContractRevisionID: "contract-2", AcceptedArtifactVersion: version}},
		{"acceptance naming a different artifact version", ExportRequest{Decision: ownerDecision("contract-1"), ContractRevisionID: "contract-1", AcceptedArtifactVersion: "sha256:another"}},
		{"acceptance with no reviewed artifact version", ExportRequest{Decision: ownerDecision("contract-1"), ContractRevisionID: "contract-1"}},
		{"acceptance with no Contract revision identity", ExportRequest{Decision: ownerDecision("contract-1"), AcceptedArtifactVersion: version}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			req.Receipt = result.Receipt
			req.Destination = filepath.Join(t.TempDir(), "export")
			if _, err := store.Export(context.Background(), req); err == nil {
				t.Fatal("export accepted an unbound acceptance decision")
			}
			if entries, err := os.ReadDir(req.Destination); err == nil && len(entries) != 0 {
				t.Fatal("refused export wrote to the destination")
			}
		})
	}
	destination := filepath.Join(t.TempDir(), "export")
	manifest, err := store.Export(context.Background(), ExportRequest{
		Receipt: result.Receipt, Decision: ownerDecision("contract-1"), ContractRevisionID: "contract-1",
		AcceptedArtifactVersion: version, Destination: destination,
	})
	if err != nil {
		t.Fatalf("exactly-bound acceptance was refused: %v", err)
	}
	if manifest.ArtifactVersion != version || manifest.AcceptanceDecisionID != "accept-1" || manifest.ContractRevisionID != "contract-1" {
		t.Fatalf("manifest does not bind the accepted result: %#v", manifest)
	}
	var written ExportManifest
	body, err := os.ReadFile(filepath.Join(destination, manifestName))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &written); err != nil {
		t.Fatal(err)
	}
	if written.ArtifactVersion != version {
		t.Fatalf("delivered manifest = %#v", written)
	}
}

// ST4. Porcelain v1 -z separates status from path with exactly one space.
// Every byte after it is the filename, including legal leading and trailing
// spaces; trimming retains the wrong path.
func TestParsePorcelainPreservesPathBytes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry string
		want  string
		kind  domain.ArtifactChangeKind
	}{
		{"leading and trailing spaces", "??  report.txt ", " report.txt ", domain.ArtifactUntracked},
		{"ordinary path", "?? report.txt", "report.txt", domain.ArtifactUntracked},
		{"modified with a spaced name", " M  spaced name.md", " spaced name.md", domain.ArtifactModified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			paths := map[string]domain.ArtifactChangeKind{}
			parsePorcelain(tc.entry+"\x00", paths)
			kind, ok := paths[tc.want]
			if !ok {
				t.Fatalf("parsed %v, want key %q", paths, tc.want)
			}
			if kind != tc.kind {
				t.Fatalf("kind = %s, want %s", kind, tc.kind)
			}
		})
	}
}

// The same defect through the real Git path: a tracked filename with a
// trailing space must be retained under its actual name.
func TestRetainPreservesGitPathsWithSurroundingSpaces(t *testing.T) {
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Kennel Test")
	if err := os.WriteFile(filepath.Join(repo, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-qm", "base")
	base := git(t, repo, "rev-parse", "HEAD")

	spaced := "report .txt "
	if err := os.WriteFile(filepath.Join(repo, spaced), []byte("spaced output\n"), 0o644); err != nil {
		t.Skipf("filesystem rejects trailing-space filenames: %v", err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), Input{
		AttemptID: "spaced-attempt", OutcomeID: "outcome-1", PlanRevisionID: "plan-1", WorkUnitID: "unit-1",
		ContractRevisionNumber: 1, WorkspaceKind: domain.WorkspaceGitWorktree, WorkspacePath: repo, BaseRevision: base,
	})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, file := range result.Receipt.Files {
		if file.RelativePath == spaced {
			found = true
		}
	}
	if !found {
		var got []string
		for _, file := range result.Receipt.Files {
			got = append(got, file.RelativePath)
		}
		t.Fatalf("retained %v, want the exact path %q", got, spaced)
	}
}

// ST5. Publication must flush the content, not only the top staging
// directory, so a published manifest version cannot name bytes that were never
// durable. This asserts the ordering contract; it does not simulate power loss.
func TestPublishedNestedContentIsFlushedBeforePublication(t *testing.T) {
	workspace := t.TempDir()
	nested := filepath.Join(workspace, "deep", "nested")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "result.txt"), []byte("nested output\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), stagedInput("nested-attempt", workspace))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Receipt.RetentionState.Complete() {
		t.Fatalf("state = %s", result.Receipt.RetentionState)
	}
	// syncTree must reach every directory it published, deepest first.
	if err := syncTree(result.ContentDir); err != nil {
		t.Fatalf("published tree is not syncable: %v", err)
	}
	content, _, err := store.Read(context.Background(), result.Receipt, result.Receipt.Files[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "nested output\n" {
		t.Fatalf("content = %q", content)
	}
}
