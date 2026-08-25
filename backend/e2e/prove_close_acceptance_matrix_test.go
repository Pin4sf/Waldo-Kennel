// DRAFT v2 — DeepSeek lane #38 evaluation fixtures. Compile-exact against
// beta @ 601c64ca5 harness + DTOs. Do not push until the owned-files boundary
// is recorded on #38. Target path:
// backend/e2e/prove_close_acceptance_matrix_test.go (package e2e).

//go:build !windows

package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

// Independent gate: these fixtures drive only outcome/proof endpoints — no
// agent spawn, no model calls, no codex binary needed.
const proofGateEnv = "KENNEL_PROOF_E2E"

func requireProofE2E(t *testing.T) {
	t.Helper()
	if os.Getenv(proofGateEnv) != "1" {
		t.Skipf("set %s=1 to run the prove/close black-box matrix", proofGateEnv)
	}
}

type proofCriterion struct {
	CriterionID        string `json:"criterionId"`
	ContractRevisionID string `json:"contractRevisionId"`
}

type proofDecision struct {
	ID                  string `json:"id"`
	ContractRevisionID  string `json:"contractRevisionId"`
	Kind                string `json:"kind"`
	ActorType           string `json:"actorType"`
	Summary             string `json:"summary"`
	ResourceDisposition string `json:"resourceDisposition"`
}

type proofView struct {
	Status     string               `json:"status"`
	NextAction string               `json:"nextAction"`
	Criteria   []proofCriterionView `json:"criteria"`
	Decisions  []proofDecision      `json:"decisions"`
}

type proofCriterionView struct {
	CriterionID   string           `json:"criterionId"`
	Ready         bool             `json:"ready"`
	Gap           string           `json:"gap"`
	Evidence      []map[string]any `json:"evidence"`
	Verifications []map[string]any `json:"verifications"`
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// seedProofOutcome creates project + outcome(rev1, two stable criteria) through
// the public API and returns ids needed by the proof endpoints.
func seedProofOutcome(t *testing.T, d *daemon) (outcomeID, revID string, revNum int64, c1, c2 proofCriterion) {
	t.Helper()
	projectID := seedProject(t, d, "proof-matrix")
	var created struct {
		Outcome struct {
			ID              string `json:"id"`
			CurrentRevision struct {
				ID       string           `json:"id"`
				Number   int64            `json:"number"`
				Criteria []proofCriterion `json:"criteria"`
			} `json:"currentRevision"`
		} `json:"outcome"`
	}
	d.mustCall("POST", "/projects/"+projectID+"/outcomes", 201, map[string]any{
		"title":           "Local Focus Ledger",
		"goal":            "Record and retain focus time.",
		"successCriteria": []string{"One block records.", "Total survives restart."},
		"review":          "Deterministic checks plus owner walkthrough.",
		"requestKey":      "seed-proof-" + t.Name(),
	}, &created)
	o := created.Outcome
	if len(o.CurrentRevision.Criteria) != 2 {
		t.Fatalf("expected two criteria, got %d", len(o.CurrentRevision.Criteria))
	}
	return o.ID, o.CurrentRevision.ID, o.CurrentRevision.Number,
		o.CurrentRevision.Criteria[0], o.CurrentRevision.Criteria[1]
}

func criterionByID(pv proofView, id string) *proofCriterionView {
	for i := range pv.Criteria {
		if pv.Criteria[i].CriterionID == id {
			return &pv.Criteria[i]
		}
	}
	return nil
}

func recordEvidence(t *testing.T, d *daemon, outcomeID string, revNum int64, revID, criterionID, key string) string {
	t.Helper()
	d.mustCall("POST", "/outcomes/"+outcomeID+"/evidence", 201, map[string]any{
		"expectedContractRevision": revNum,
		"contractRevisionId":       revID,
		"criterionId":              criterionID,
		"subjectType":              "outcome",
		"subjectId":                outcomeID,
		"subjectRevision":          revID,
		"kind":                     "supporting",
		"sourceType":               "deterministic_check",
		"sourceRef":                "check-" + key,
		"producerType":             "tool",
		"producerRef":              "go-test",
		"summary":                  "deterministic check output for " + key,
		"contentDigest":            digest(key),
		"requestKey":               key,
	}, nil)
	pv := getProof(t, d, outcomeID)
	c := criterionByID(pv, criterionID)
	if c == nil || len(c.Evidence) == 0 {
		t.Fatalf("evidence %s missing after record", key)
	}
	id, _ := c.Evidence[len(c.Evidence)-1]["id"].(string)
	return id
}

func recordVerification(t *testing.T, d *daemon, outcomeID string, revNum int64, revID, criterionID, evidenceID, result, key string) {
	t.Helper()
	d.mustCall("POST", "/outcomes/"+outcomeID+"/verifications", 201, map[string]any{
		"expectedContractRevision": revNum,
		"contractRevisionId":       revID,
		"criterionId":              criterionID,
		"subjectType":              "outcome",
		"subjectId":                outcomeID,
		"subjectRevision":          revID,
		"evidenceItemIds":          []string{evidenceID},
		"method":                   "go test ./...",
		"independenceClass":        "separate_session",
		"result":                   result,
		"producerRef":              "go-test",
		"verifierRef":              "fresh-verifier",
		"requestKey":               key,
	}, nil)
}

func getProof(t *testing.T, d *daemon, outcomeID string) proofView {
	t.Helper()
	var env struct {
		Proof proofView `json:"proof"`
	}
	d.mustCall("GET", "/outcomes/"+outcomeID+"/proof", 200, nil, &env)
	return env.Proof
}

func decide(t *testing.T, d *daemon, outcomeID string, revNum int64, revID string, body map[string]any) {
	t.Helper()
	d.mustCall("POST", "/outcomes/"+outcomeID+"/acceptance-decisions", 201, body, nil)
}

// Scenario 1 — deterministic verification failure keeps the Outcome open;
// acceptance stays unavailable; rework requires explicit re-entry lineage.
func TestProofMatrixVerificationFailureKeepsOutcomeOpen(t *testing.T) {
	requireProofE2E(t)
	d := startDaemon(t, t.TempDir())
	defer d.stop()
	outcomeID, revID, revNum, c1, c2 := seedProofOutcome(t, d)

	e1 := recordEvidence(t, d, outcomeID, revNum, revID, c1.CriterionID, "ev-ok")
	recordVerification(t, d, outcomeID, revNum, revID, c1.CriterionID, e1, "passed", "ver-ok")
	e2 := recordEvidence(t, d, outcomeID, revNum, revID, c2.CriterionID, "ev-bad")
	recordVerification(t, d, outcomeID, revNum, revID, c2.CriterionID, e2, "failed", "ver-bad")

	proof := getProof(t, d, outcomeID)
	if proof.Status == "ready_for_acceptance" || proof.Status == "accepted" {
		t.Fatalf("failed verification must not ready the Outcome: status=%q", proof.Status)
	}
	status, apiErr := d.callExpectingError("POST", "/outcomes/"+outcomeID+"/acceptance-decisions", map[string]any{
		"expectedContractRevision": revNum, "contractRevisionId": revID,
		"kind": "accept", "summary": "premature", "resourceDisposition": "retain",
		"requestKey": "acc-premature",
	})
	if status != 409 || apiErr.Code != "OUTCOME_NOT_READY_FOR_ACCEPTANCE" {
		t.Fatalf("want 409 OUTCOME_NOT_READY_FOR_ACCEPTANCE, got %d %+v", status, apiErr)
	}
	if p := getProof(t, d, outcomeID); len(p.Decisions) != 0 {
		t.Fatalf("refused acceptance must not append a decision: %+v", p.Decisions)
	}
	decide(t, d, outcomeID, revNum, revID, map[string]any{
		"expectedContractRevision": revNum, "contractRevisionId": revID,
		// No plan exists in this minimal lineage, so re-entry targets the
		// current contract itself (canonical spec: correction may lead to a
		// replacement Attempt, revised WorkUnit/Plan, or revised Contract).
		"kind": "request_rework", "summary": "criterion two failed deterministically",
		"resourceDisposition": "not_applicable",
		"reentryTargetType":   "contract", "reentryTargetID": revID,
		"requestKey": "rework-1",
	})
	if p := getProof(t, d, outcomeID); p.Status != "rework_required" {
		t.Fatalf("want rework_required, got %q", p.Status)
	}
}

// Scenario 2 — explicit acceptance then reopen survive a real daemon restart
// against the SAME data dir; decision history is immutable and append-only.
func TestProofMatrixAcceptReopenDurableAcrossRestart(t *testing.T) {
	requireProofE2E(t)
	dataDir := t.TempDir()
	d := startDaemon(t, dataDir)
	outcomeID, revID, revNum, c1, c2 := seedProofOutcome(t, d)
	for _, c := range []proofCriterion{c1, c2} {
		e := recordEvidence(t, d, outcomeID, revNum, revID, c.CriterionID, "ev-"+c.CriterionID)
		recordVerification(t, d, outcomeID, revNum, revID, c.CriterionID, e, "passed", "ver-"+c.CriterionID)
	}
	decide(t, d, outcomeID, revNum, revID, map[string]any{
		"expectedContractRevision": revNum, "contractRevisionId": revID,
		"kind": "accept", "summary": "owner accepts the ledger",
		"resourceDisposition": "retain", "requestKey": "accept-1",
	})
	before := getProof(t, d, outcomeID)
	if before.Status != "accepted" || len(before.Decisions) != 1 {
		t.Fatalf("want single accepted decision, got %q n=%d", before.Status, len(before.Decisions))
	}

	d.stop() // real stop; restart on the same data dir below
	d = startDaemon(t, dataDir)
	defer d.stop()

	after := getProof(t, d, outcomeID)
	if after.Status != "accepted" || len(after.Decisions) != 1 {
		t.Fatalf("restart lost accepted state: %q n=%d", after.Status, len(after.Decisions))
	}
	if after.Decisions[0] != before.Decisions[0] {
		t.Fatalf("decision mutated across restart:\nbefore %+v\nafter  %+v", before.Decisions[0], after.Decisions[0])
	}
	decide(t, d, outcomeID, revNum, revID, map[string]any{
		"expectedContractRevision": revNum, "contractRevisionId": revID,
		"kind": "reopen", "summary": "total was wrong after midnight",
		"resourceDisposition": "not_applicable",
		"reentryTargetType":   "contract", "reentryTargetID": revID,
		"requestKey": "reopen-1",
	})
	final := getProof(t, d, outcomeID)
	if final.Status != "rework_required" || len(final.Decisions) != 2 {
		t.Fatalf("reopen wrong: %q n=%d", final.Status, len(final.Decisions))
	}
	if final.Decisions[0].Kind != "accept" || final.Decisions[0].ID != before.Decisions[0].ID {
		t.Fatalf("prior decision not preserved append-only: %+v", final.Decisions[0])
	}
}

// Scenario 3 — provider/session/check/test completion facts can NEVER create an
// AcceptanceDecision. Provider-sourced Evidence may exist; decisions appear
// only through the explicit user endpoint, gated on readiness.
func TestProofMatrixProviderFactsNeverCreateAcceptance(t *testing.T) {
	requireProofE2E(t)
	d := startDaemon(t, t.TempDir())
	defer d.stop()
	outcomeID, revID, revNum, c1, c2 := seedProofOutcome(t, d)

	e := recordEvidence(t, d, outcomeID, revNum, revID, c1.CriterionID, "ev-provider")
	recordVerification(t, d, outcomeID, revNum, revID, c1.CriterionID, e, "passed", "ver-provider")

	mid := getProof(t, d, outcomeID)
	if mid.Status == "accepted" || len(mid.Decisions) != 0 {
		t.Fatalf("completion-ish facts minted decisions: %q %+v", mid.Status, mid.Decisions)
	}
	// Second criterion still unproven: readiness incomplete, acceptance refused.
	status, _ := d.callExpectingError("POST", "/outcomes/"+outcomeID+"/acceptance-decisions", map[string]any{
		"expectedContractRevision": revNum, "contractRevisionId": revID,
		"kind": "accept", "summary": "partial", "resourceDisposition": "retain",
		"requestKey": "acc-partial",
	})
	if status != 409 {
		t.Fatalf("partial readiness must refuse acceptance, got %d", status)
	}
	_ = c2 // criterion two intentionally never proven in this scenario
}
