package outcome_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/artifactstore"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// deliverThenLoseTheLedger performs a real delivery and then rewinds its row to
// pending — the state a crash between the filesystem commit and the ledger
// write leaves behind. The bytes are genuinely at the destination; only the
// record of them is gone.
func deliverThenLoseTheLedger(t *testing.T, h *deliveryReviewHarness, destination, key string) domain.OutcomeDelivery {
	t.Helper()
	delivered, err := h.svc.RequestDelivery(context.Background(), h.outcomeID, h.request(destination, key))
	if err != nil {
		t.Fatalf("request delivery: %v", err)
	}
	if delivered.State != domain.DeliverySucceeded {
		t.Fatalf("state = %q, want succeeded before the ledger is lost", delivered.State)
	}
	if delivered.CompletionSource != domain.DeliveryObserved {
		t.Fatalf("completion source = %q, want observed for a watched transfer", delivered.CompletionSource)
	}
	pending := delivered
	pending.State, pending.CompletedAt, pending.ManifestPath = domain.DeliveryPending, nil, ""
	pending.FileCount, pending.ByteCount, pending.CompletionSource = 0, 0, ""
	h.deliveries.byID[pending.ID], h.deliveries.byKey[pending.RequestKey] = pending, pending
	return delivered
}

func (h *deliveryReviewHarness) reload(t *testing.T, id domain.DeliveryID) domain.OutcomeDelivery {
	t.Helper()
	delivery, err := h.svc.GetDelivery(context.Background(), h.outcomeID, id)
	if err != nil {
		t.Fatalf("read delivery %s: %v", id, err)
	}
	return delivery
}

// TestReconcileDeliveries_RecoversACommittedTransferWhoseLedgerWriteWasLost is
// DLV-01. The transfer completed and the row never reached a terminal state, so
// recovery reads the destination, matches it against the authorized delivery,
// and records the completion it can prove — labelled as established by reading
// rather than by watching.
func TestReconcileDeliveries_RecoversACommittedTransferWhoseLedgerWriteWasLost(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	destination := filepath.Join(t.TempDir(), "restart-bundle")
	delivered := deliverThenLoseTheLedger(t, h, destination, "dlv-restart")

	recovery, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if recovery.Recovered != 1 || recovery.Interrupted != 0 || recovery.Ambiguous != 0 {
		t.Fatalf("recovery = %+v, want exactly one recovered transfer", recovery)
	}

	after := h.reload(t, delivered.ID)
	if after.State != domain.DeliverySucceeded {
		t.Fatalf("state = %q/%q, want the completed transfer recorded", after.State, after.FailureCode)
	}
	if after.CompletionSource != domain.DeliveryRecovered {
		t.Fatalf("completion source = %q, want recovered: nobody watched this transfer", after.CompletionSource)
	}
	if after.ManifestPath != filepath.Join(destination, artifactstore.ManifestName) {
		t.Fatalf("manifest path = %q", after.ManifestPath)
	}
	if after.FileCount != delivered.FileCount || after.ByteCount != delivered.ByteCount {
		t.Fatalf("recovered size = %d files / %d bytes, want %d / %d",
			after.FileCount, after.ByteCount, delivered.FileCount, delivered.ByteCount)
	}
	// Recovery proves the transfer, never acceptance. The disposition is the
	// only thing that speaks to acceptance and it is unchanged.
	if after.Disposition != delivered.Disposition || after.AcceptanceDecisionID != delivered.AcceptanceDecisionID {
		t.Fatalf("recovery changed the acceptance binding: %+v", after)
	}
}

// TestReconcileDeliveries_IsIdempotentAcrossRepeatedPasses keeps a second
// reconciliation from re-deciding a row it already closed, which is what a
// daemon restarted twice in a row would do.
func TestReconcileDeliveries_IsIdempotentAcrossRepeatedPasses(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	destination := filepath.Join(t.TempDir(), "twice-bundle")
	delivered := deliverThenLoseTheLedger(t, h, destination, "dlv-twice")

	first, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if first.Recovered != 1 {
		t.Fatalf("first pass = %+v, want one recovered", first)
	}
	second, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if second.Closed() != 0 {
		t.Fatalf("second pass = %+v, want nothing left to close", second)
	}
	if after := h.reload(t, delivered.ID); after.State != domain.DeliverySucceeded ||
		after.CompletionSource != domain.DeliveryRecovered {
		t.Fatalf("delivery = %+v after a repeated pass", after)
	}
}

// TestReconcileDeliveries_KeepsUnprovableDestinationsExplicit is the whole
// safety argument. Only an exact match may be recorded as a completed transfer;
// everything else stays a distinct, explicit failure, and none of it writes to
// the destination or re-transfers.
func TestReconcileDeliveries_KeepsUnprovableDestinationsExplicit(t *testing.T) {
	cases := []struct {
		name string
		// disturb mutates the destination after a real delivery, modelling what
		// reconciliation might find there.
		disturb  func(t *testing.T, destination string)
		wantCode string
		wantKind string
	}{
		{
			name: "nothing was transferred",
			disturb: func(t *testing.T, destination string) {
				if err := os.RemoveAll(destination); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: "DELIVERY_INTERRUPTED", wantKind: "interrupted",
		},
		{
			name: "a delivered artifact is missing",
			disturb: func(t *testing.T, destination string) {
				if err := os.Remove(filepath.Join(destination, "answer.txt")); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: "DELIVERY_RECOVERY_MISMATCH", wantKind: "ambiguous",
		},
		{
			name: "a delivered artifact no longer matches its digest",
			disturb: func(t *testing.T, destination string) {
				if err := os.WriteFile(filepath.Join(destination, "answer.txt"), []byte("tampered\n"), 0o640); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: "DELIVERY_RECOVERY_MISMATCH", wantKind: "ambiguous",
		},
		{
			name: "the destination holds unrelated content with no manifest",
			disturb: func(t *testing.T, destination string) {
				if err := os.Remove(filepath.Join(destination, artifactstore.ManifestName)); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: "DELIVERY_RECOVERY_MISMATCH", wantKind: "ambiguous",
		},
		{
			name: "the manifest cannot be parsed",
			disturb: func(t *testing.T, destination string) {
				if err := os.WriteFile(filepath.Join(destination, artifactstore.ManifestName), []byte("{not json"), 0o640); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: "DELIVERY_RECOVERY_UNREADABLE", wantKind: "ambiguous",
		},
		{
			name: "the manifest describes a different delivery",
			disturb: func(t *testing.T, destination string) {
				const other = `{"attemptId":"attempt-somebody-else","outcomeId":"outcome-review",` +
					`"artifactVersion":"v-other","disposition":"accepted","files":[]}`
				if err := os.WriteFile(filepath.Join(destination, artifactstore.ManifestName), []byte(other), 0o640); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: "DELIVERY_RECOVERY_MISMATCH", wantKind: "ambiguous",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newDeliveryReviewHarness(t)
			destination := filepath.Join(t.TempDir(), "disturbed-bundle")
			delivered := deliverThenLoseTheLedger(t, h, destination, "dlv-disturbed")
			tc.disturb(t, destination)

			before := snapshotTree(t, destination)
			recovery, err := h.svc.ReconcileDeliveries(context.Background())
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			switch tc.wantKind {
			case "interrupted":
				if recovery.Interrupted != 1 || recovery.Recovered != 0 || recovery.Ambiguous != 0 {
					t.Fatalf("recovery = %+v, want one interrupted", recovery)
				}
			default:
				if recovery.Ambiguous != 1 || recovery.Recovered != 0 || recovery.Interrupted != 0 {
					t.Fatalf("recovery = %+v, want one ambiguous", recovery)
				}
			}

			after := h.reload(t, delivered.ID)
			if after.State != domain.DeliveryFailed {
				t.Fatalf("state = %q, want failed: an unprovable transfer is never recorded as done", after.State)
			}
			if after.FailureCode != tc.wantCode {
				t.Fatalf("failure code = %q, want %q (%s)", after.FailureCode, tc.wantCode, after.FailureDetail)
			}
			if after.FailureDetail == "" {
				t.Fatal("an explicit failure with no reason cannot be acted on")
			}
			// Recovery reads. It must not re-transfer, replace or remove
			// anything — including whatever somebody else's files turned out
			// to be.
			if got := snapshotTree(t, destination); got != before {
				t.Fatalf("recovery changed the destination:\nbefore %s\nafter  %s", before, got)
			}
		})
	}
}

// TestReconcileDeliveries_CannotVerifyWithoutTheRetainedResult keeps recovery
// from trusting a destination to describe itself. With the retained result
// gone there is nothing authoritative to compare against, so the row stays an
// explicit failure rather than a success taken from a manifest alone.
func TestReconcileDeliveries_CannotVerifyWithoutTheRetainedResult(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	destination := filepath.Join(t.TempDir(), "unverifiable-bundle")
	delivered := deliverThenLoseTheLedger(t, h, destination, "dlv-unverifiable")

	// The delivery row still names an artifact version the receipt no longer
	// reports, which is what a replaced or re-retained result looks like.
	pending := h.deliveries.byID[delivered.ID]
	pending.ArtifactVersion = "v-no-longer-retained"
	h.deliveries.byID[pending.ID], h.deliveries.byKey[pending.RequestKey] = pending, pending

	recovery, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if recovery.Ambiguous != 1 {
		t.Fatalf("recovery = %+v, want one ambiguous", recovery)
	}
	after := h.reload(t, delivered.ID)
	if after.State != domain.DeliveryFailed || after.FailureCode != "DELIVERY_RECOVERY_UNVERIFIABLE" {
		t.Fatalf("delivery = %+v, want failed/DELIVERY_RECOVERY_UNVERIFIABLE", after)
	}
}

// TestReconcileDeliveries_RecoversATransferThroughASymlinkedParent pins the
// read boundary against DLV-02. The export follows a symlinked parent, so the
// bytes really are behind the link; verification has to look in the same place
// or it would report a completed transfer as missing.
func TestReconcileDeliveries_RecoversATransferThroughASymlinkedParent(t *testing.T) {
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
	delivered := deliverThenLoseTheLedger(t, h, filepath.Join(link, "bundle"), "dlv-linked")

	recovery, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if recovery.Recovered != 1 {
		t.Fatalf("recovery = %+v, want the linked transfer recovered", recovery)
	}
	if after := h.reload(t, delivered.ID); after.State != domain.DeliverySucceeded {
		t.Fatalf("delivery = %+v, want succeeded", after)
	}
}

// TestVerifyExport_RefusesASymlinkStandingInForADeliveredArtifact keeps a link
// inside the delivered tree from making an unrelated file read as delivered
// content, which would turn a mismatch into a false match.
func TestVerifyExport_RefusesASymlinkStandingInForADeliveredArtifact(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	destination := filepath.Join(t.TempDir(), "swapped-bundle")
	delivered := deliverThenLoseTheLedger(t, h, destination, "dlv-swapped")

	// Replace the delivered file with a link to an identical file outside the
	// destination. The bytes match; the delivered artifact does not exist.
	outside := filepath.Join(t.TempDir(), "answer.txt")
	if err := os.WriteFile(outside, []byte("accepted\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(destination, "answer.txt")
	if err := os.Remove(inside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, inside); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	recovery, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if recovery.Ambiguous != 1 {
		t.Fatalf("recovery = %+v, want the swapped artifact reported ambiguous", recovery)
	}
	after := h.reload(t, delivered.ID)
	if after.State != domain.DeliveryFailed || after.FailureCode != "DELIVERY_RECOVERY_MISMATCH" {
		t.Fatalf("delivery = %+v, want failed/DELIVERY_RECOVERY_MISMATCH", after)
	}
}

// TestVerifyExport_RefusesASymlinkedDirectoryInsideTheDestination is the
// confinement hole a leaf check cannot see.
//
// confinedPath is lexical, so a component *between* the destination and an
// artifact escapes it: `dest/nested` replaced by a link to another directory
// still joins to a path under `dest`, and the final component found through it
// is a perfectly ordinary regular file. Verification would then confirm a
// delivery whose artifacts are not at the destination at all — and would do so
// for content nobody delivered, since identical bytes anywhere satisfy the
// digest.
//
// Following a symlinked *parent of the destination* stays deliberate: the
// export wrote through it, so reading through it is how the bytes are found.
// Inside the delivered tree the export creates only real directories and
// regular files, so any link there is not ours.
func TestVerifyExport_RefusesASymlinkedDirectoryInsideTheDestination(t *testing.T) {
	h := newDeliveryReviewHarness(t)
	destination := filepath.Join(t.TempDir(), "escaped-bundle")
	delivered := deliverThenLoseTheLedger(t, h, destination, "dlv-escaped")

	// An identical copy of the nested artifact, outside the destination.
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(outside, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "deep.txt"), []byte("deep\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(destination, "nested")
	if err := os.RemoveAll(nested); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, nested); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	recovery, err := h.svc.ReconcileDeliveries(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if recovery.Recovered != 0 {
		t.Fatalf("recovery = %+v, want nothing confirmed through a symlinked directory", recovery)
	}
	after := h.reload(t, delivered.ID)
	if after.State != domain.DeliveryFailed || after.FailureCode != "DELIVERY_RECOVERY_MISMATCH" {
		t.Fatalf("delivery = %+v, want failed/DELIVERY_RECOVERY_MISMATCH", after)
	}
	if after.CompletionSource == domain.DeliveryRecovered && after.State == domain.DeliverySucceeded {
		t.Fatal("a delivery was confirmed through a directory pointing outside the destination")
	}
}

// snapshotTree renders a destination's entries and contents so a test can
// prove recovery did not touch them.
//
// Every read failure is recorded rather than raised: a destination that cannot
// be walked is itself part of what the snapshot is comparing, and an absent
// destination is a legitimate state here.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var out []byte
	record := func(format string, args ...any) {
		out = append(out, fmt.Sprintf(format, args...)...)
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			record("%s|<unwalkable: %v>\n", path, walkErr)
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			record("%s|<unrelative: %v>\n", path, relErr)
			return nil
		}
		if !info.Mode().IsRegular() {
			record("%s|<%s>\n", rel, info.Mode().Type())
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			record("%s|<unreadable: %v>\n", rel, readErr)
			return nil
		}
		record("%s|%s\n", rel, body)
		return nil
	})
	return string(out)
}
