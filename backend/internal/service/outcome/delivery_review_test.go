package outcome_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/artifactstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	outcomevc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/outcome"
)

// deliveryReviewHarness is the fixture from the delivery test, reshaped so a
// review can vary the destination and the request context.
type deliveryReviewHarness struct {
	svc        *outcomevc.Service
	deliveries *memoryDeliveryStore
	outcomeID  domain.OutcomeID
	attemptID  domain.AttemptID
	version    string
	decisionID domain.AcceptanceDecisionID
}

func newDeliveryReviewHarness(t *testing.T) *deliveryReviewHarness {
	t.Helper()
	ctx := context.Background()
	at := time.Unix(500, 0).UTC()
	artifacts, err := artifactstore.New(artifactstore.Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "answer.txt"), []byte("accepted\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	attempt := domain.Attempt{
		ID: "attempt-review", OutcomeID: "outcome-review", WorkUnitID: "unit-review",
		PlanRevisionID: "plan-review", ContractRevisionNumber: 1,
	}
	retained, err := artifacts.Retain(ctx, artifactstore.Input{
		AttemptID: attempt.ID, OutcomeID: attempt.OutcomeID, PlanRevisionID: attempt.PlanRevisionID,
		WorkUnitID: attempt.WorkUnitID, ContractRevisionNumber: 1,
		WorkspaceKind: domain.WorkspaceStagedFolder, WorkspacePath: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.AcceptanceDecision{
		ID: "accept-review", OutcomeID: attempt.OutcomeID, ContractRevisionID: "contract-review",
		Kind: domain.AcceptanceAccept, ActorType: domain.AcceptanceActorUser, Summary: "Reviewed exact result",
		ResourceDisposition: domain.ResourceDispositionRetain, RequestKey: "accept-review-key",
		RequestFingerprint: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", CreatedAt: at,
	}
	fixture := &deliveryFixture{
		outcome: domain.Outcome{ID: attempt.OutcomeID}, attempt: attempt,
		revision: domain.ContractRevision{ID: decision.ContractRevisionID, OutcomeID: attempt.OutcomeID, Number: 1},
		receipt:  retained.Receipt, decisions: []domain.AcceptanceDecision{decision},
	}
	deliveries := &memoryDeliveryStore{byKey: map[string]domain.OutcomeDelivery{}, byID: map[domain.DeliveryID]domain.OutcomeDelivery{}}
	return &deliveryReviewHarness{
		svc: outcomevc.New(fixture, func() time.Time { return at }).
			WithProofStore(fixture).WithDelivery(deliveries, artifacts),
		deliveries: deliveries, outcomeID: attempt.OutcomeID, attemptID: attempt.ID,
		version: retained.Receipt.ArtifactVersion, decisionID: decision.ID,
	}
}

func (h *deliveryReviewHarness) request(destination, key string) outcomevc.RequestDeliveryInput {
	return outcomevc.RequestDeliveryInput{
		AttemptID: h.attemptID, ArtifactVersion: h.version, Destination: destination,
		Disposition: domain.DeliveryAccepted, AcceptanceDecisionID: h.decisionID, RequestKey: key,
	}
}

// TestRequestDelivery_ACancelledRequestLeavesNothingAtTheDestination is the
// good half of the interruption boundary, and worth pinning because the rest of
// this file is about where the same boundary leaks.
//
// The export stages into a sibling directory, verifies every file against the
// retained digest, and commits by renaming. A cancelled request therefore
// leaves no partial bundle for an owner to mistake for a result — the artifact
// reads honour the context, so the staging directory is removed and the
// destination is never created.
func TestRequestDelivery_ACancelledRequestLeavesNothingAtTheDestination(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	destination := filepath.Join(t.TempDir(), "cancelled-bundle")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	delivery, err := h.svc.RequestDelivery(ctx, h.outcomeID, h.request(destination, "dlv-cancelled"))
	if err != nil {
		t.Fatalf("request delivery: %v", err)
	}
	if delivery.State != domain.DeliveryCancelled {
		t.Fatalf("state = %q, want cancelled", delivery.State)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("destination stat = %v, want a cancelled delivery to leave nothing behind", statErr)
	}
	// No staging debris beside it either: a half-written export must not be
	// left where the next request would collide with it.
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging debris left behind: %v", entries)
	}
	// And the owner can simply retry, because nothing is in the way.
	retry, err := h.svc.RequestDelivery(context.Background(), h.outcomeID,
		h.request(destination, "dlv-cancelled-retry"))
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retry.State != domain.DeliverySucceeded {
		t.Fatalf("retry = %+v, want a clean retry to succeed", retry)
	}
}

// TestRequestDelivery_ASymlinkedParentIsNotRefusedTheWayALeafSymlinkIs
// characterises the destination-safety boundary. The final path component is
// checked twice — by the service and again by the export — and a symlink there
// is refused. No component above it is checked, so a symlinked parent silently
// redirects the whole transfer.
func TestRequestDelivery_ASymlinkedParentIsNotRefusedTheWayALeafSymlinkIs(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	root := t.TempDir()
	elsewhere := filepath.Join(root, "elsewhere")
	if err := os.Mkdir(elsewhere, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "exports")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	// A symlink AS the destination is refused, and that refusal is what makes
	// the parent case a gap rather than a deliberate policy.
	if _, err := h.svc.RequestDelivery(context.Background(), h.outcomeID,
		h.request(link, "dlv-leaf-symlink")); err == nil {
		t.Fatal("a symlink destination was accepted")
	}

	delivery, err := h.svc.RequestDelivery(context.Background(), h.outcomeID,
		h.request(filepath.Join(link, "bundle"), "dlv-parent-symlink"))
	if err != nil {
		t.Fatalf("request delivery through a symlinked parent: %v", err)
	}
	if delivery.State != domain.DeliverySucceeded {
		t.Fatalf("state = %q, want succeeded — this test records current behaviour", delivery.State)
	}
	// The bytes landed outside the path the owner named, by way of the link.
	if _, statErr := os.Stat(filepath.Join(elsewhere, "bundle", "answer.txt")); statErr != nil {
		t.Fatalf("resolved destination = %v, want the transfer to have followed the link", statErr)
	}
}
