package outcome_test

import (
	"context"
	"sync"
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// receiptFakeStore adds retained receipts and the success finalizer to the
// attempt fake, so terminal classification is exercised against real state
// transitions rather than mocks.
//
// The finalizer reproduces the two checks the SQLite commit actually makes —
// expected status and observed proof generation — because those are what the
// service's read ordering has to survive. A finalizer that accepted anything
// would make the ordering untestable.
type receiptFakeStore struct {
	*attemptFakeStore

	receiptMu sync.Mutex
	artifacts map[domain.AttemptID]domain.AttemptReceipt

	// afterEvidenceRead fires once, immediately after an evidence read
	// returns. It is the deterministic stand-in for a record committing just
	// after a proof snapshot was taken.
	afterEvidenceRead func()
	evidenceReadFired bool

	// readOrder records which of the two classification reads happened first.
	readOrder []string
}

func newReceiptFakeStore() *receiptFakeStore {
	return &receiptFakeStore{
		attemptFakeStore: newAttemptFakeStore(),
		artifacts:        map[domain.AttemptID]domain.AttemptReceipt{},
	}
}

func (f *receiptFakeStore) SaveAttemptReceipt(_ context.Context, receipt domain.AttemptReceipt) error {
	f.receiptMu.Lock()
	defer f.receiptMu.Unlock()
	if existing, ok := f.artifacts[receipt.AttemptID]; ok && existing.Frozen() {
		return ports.ErrAttemptReceiptFrozen
	}
	f.artifacts[receipt.AttemptID] = receipt
	return nil
}

func (f *receiptFakeStore) GetAttemptReceipt(_ context.Context, id domain.AttemptID) (domain.AttemptReceipt, bool, error) {
	f.receiptMu.Lock()
	defer f.receiptMu.Unlock()
	receipt, ok := f.artifacts[id]
	return receipt, ok, nil
}

func (f *receiptFakeStore) FreezeAttemptReceipt(_ context.Context, id domain.AttemptID, at time.Time) error {
	f.receiptMu.Lock()
	defer f.receiptMu.Unlock()
	receipt, ok := f.artifacts[id]
	if !ok {
		return ports.ErrAttemptReceiptMissing
	}
	if receipt.Frozen() {
		return nil
	}
	frozen := at.UTC()
	receipt.FrozenAt = &frozen
	f.artifacts[id] = receipt
	return nil
}

// OutcomeProofGeneration counts the append-only proof records, exactly as the
// SQLite implementation does.
func (f *receiptFakeStore) OutcomeProofGeneration(ctx context.Context, outcomeID domain.OutcomeID) (int64, error) {
	f.receiptMu.Lock()
	f.readOrder = append(f.readOrder, "generation")
	f.receiptMu.Unlock()
	return f.proofRecordCount(ctx, outcomeID)
}

func (f *receiptFakeStore) proofRecordCount(ctx context.Context, outcomeID domain.OutcomeID) (int64, error) {
	evidence, err := f.attemptFakeStore.ListEvidenceItems(ctx, outcomeID)
	if err != nil {
		return 0, err
	}
	runs, err := f.attemptFakeStore.ListVerificationRuns(ctx, outcomeID)
	if err != nil {
		return 0, err
	}
	decisions, err := f.attemptFakeStore.ListAcceptanceDecisions(ctx, outcomeID)
	if err != nil {
		return 0, err
	}
	corrections, err := f.attemptFakeStore.ListOutcomeCorrections(ctx, outcomeID)
	if err != nil {
		return 0, err
	}
	return int64(len(evidence) + len(runs) + len(decisions) + len(corrections)), nil
}

// ListEvidenceItems is one of the reads GetProof performs. The hook fires
// after the read returns, which is the window the classification ordering has
// to survive.
func (f *receiptFakeStore) ListEvidenceItems(ctx context.Context, outcomeID domain.OutcomeID) ([]domain.EvidenceItem, error) {
	items, err := f.attemptFakeStore.ListEvidenceItems(ctx, outcomeID)
	f.receiptMu.Lock()
	f.readOrder = append(f.readOrder, "proof")
	hook := f.afterEvidenceRead
	fire := hook != nil && !f.evidenceReadFired
	if fire {
		f.evidenceReadFired = true
	}
	f.receiptMu.Unlock()
	if fire {
		hook()
	}
	return items, err
}

// ClassifyAttemptSucceeded commits only if the Attempt is still in the
// expected status and the proof generation the judgement rested on has not
// moved. Those are the guards the read ordering exists to arm.
func (f *receiptFakeStore) ClassifyAttemptSucceeded(ctx context.Context, in ports.ClassifyAttemptInput) error {
	if in.ProofGeneration == nil || in.ContractRevisionNumber < 1 {
		return ports.ErrAttemptClassificationStale
	}
	current, err := f.proofRecordCount(ctx, in.OutcomeID)
	if err != nil {
		return err
	}
	if current != *in.ProofGeneration {
		return ports.ErrAttemptClassificationStale
	}
	receipt, ok, err := f.GetAttemptReceipt(ctx, in.AttemptID)
	if err != nil {
		return err
	}
	if !ok {
		return ports.ErrAttemptReceiptMissing
	}
	if receipt.ArtifactVersion != in.ArtifactVersion || !receipt.RetentionState.Complete() {
		return ports.ErrAttemptReceiptNotReady
	}
	rows, err := f.TransitionAttemptStatus(ctx, in.OutcomeID, in.AttemptID, in.ExpectedStatus, domain.AttemptSucceeded, in.At)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ports.ErrAttemptClassificationStale
	}
	if err := f.FreezeAttemptReceipt(ctx, in.AttemptID, in.At); err != nil {
		return err
	}
	_, err = f.AppendAttemptObservation(ctx, in.AttemptID, in.ObservationKind, in.ObservationPayload, in.At)
	return err
}

// resetReads clears read bookkeeping so a test measures classification's
// reads rather than its own setup's.
func (f *receiptFakeStore) resetReads() {
	f.receiptMu.Lock()
	defer f.receiptMu.Unlock()
	f.readOrder = nil
}

func (f *receiptFakeStore) firstRead() string {
	f.receiptMu.Lock()
	defer f.receiptMu.Unlock()
	if len(f.readOrder) == 0 {
		return ""
	}
	return f.readOrder[0]
}
