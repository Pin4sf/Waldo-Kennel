package outcome

import (
	"strings"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func passingObservation() ports.AttemptCheckObservation {
	return ports.AttemptCheckObservation{
		Check:      domain.ApprovedCheck{ID: "chk-1", CriterionID: "crit-a", Argv: []string{"true"}, TimeoutSeconds: 30},
		Ran:        true,
		Passed:     true,
		EnforcedBy: "macos-seatbelt-workspace-write",
	}
}

// TestCheckVerdict_DistinguishesDidNotRunFromFailed is the distinction the
// owner acts on. A host with no sandbox has a setup problem; a red check has
// a work problem. Collapsing them sends the owner to fix the wrong thing.
func TestCheckVerdict_DistinguishesDidNotRunFromFailed(t *testing.T) {
	cases := []struct {
		name            string
		observation     ports.AttemptCheckObservation
		artifactChanged bool
		wantKind        domain.EvidenceKind
		wantResult      domain.VerificationResult
	}{
		{
			name:        "a check that ran and exited zero supports the result",
			observation: passingObservation(),
			wantKind:    domain.EvidenceSupporting, wantResult: domain.VerificationPassed,
		},
		{
			name: "a check that ran and exited non-zero contradicts it",
			observation: func() ports.AttemptCheckObservation {
				o := passingObservation()
				o.Passed, o.ExitCode = false, 1
				return o
			}(),
			wantKind: domain.EvidenceContradicting, wantResult: domain.VerificationFailed,
		},
		{
			name: "a check that never launched proves nothing either way",
			observation: ports.AttemptCheckObservation{
				Check:       passingObservation().Check,
				Unavailable: "no mechanism can enforce the approved limits for this check",
			},
			wantKind: domain.EvidenceSupporting, wantResult: domain.VerificationInconclusive,
		},
		{
			name: "a check whose process tree could not be confirmed stopped is inconclusive",
			observation: func() ports.AttemptCheckObservation {
				o := passingObservation()
				o.TerminationUnknown = true
				return o
			}(),
			wantKind: domain.EvidenceSupporting, wantResult: domain.VerificationInconclusive,
		},
		{
			// A pass taken from bytes the check then rewrote is not a pass on
			// the retained result.
			name:            "a passing check that changed the result it checked is inconclusive",
			observation:     passingObservation(),
			artifactChanged: true,
			wantKind:        domain.EvidenceSupporting, wantResult: domain.VerificationInconclusive,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, result := checkVerdict(tc.observation, tc.artifactChanged)
			if kind != tc.wantKind || result != tc.wantResult {
				t.Fatalf("verdict = %s/%s, want %s/%s", kind, result, tc.wantKind, tc.wantResult)
			}
		})
	}
}

// TestCheckVerifierRef_NeverImpliesConfinementThatDidNotExist keeps evidence
// honest about what actually held the boundary.
func TestCheckVerifierRef_NeverImpliesConfinementThatDidNotExist(t *testing.T) {
	if got := checkVerifierRef(passingObservation()); got != "kennel-governed-check/macos-seatbelt-workspace-write" {
		t.Fatalf("verifier ref = %q", got)
	}
	if got := checkVerifierRef(ports.AttemptCheckObservation{}); got != "kennel-governed-check/unenforced" {
		t.Fatalf("unenforced verifier ref = %q, want it to say so", got)
	}
}

// TestCheckRequestKey_IsPerAttemptPerArtifactPerCheck is what makes a repeated
// reconciliation tick a replay rather than a second observation, while a new
// artifact version legitimately gets its own.
func TestCheckRequestKey_IsPerAttemptPerArtifactPerCheck(t *testing.T) {
	base := checkRequestKey("att-1", "v1", "chk-1")
	if base != checkRequestKey("att-1", "v1", "chk-1") {
		t.Fatal("the same observation produced two request keys")
	}
	for _, other := range []string{
		checkRequestKey("att-2", "v1", "chk-1"),
		checkRequestKey("att-1", "v2", "chk-1"),
		checkRequestKey("att-1", "v1", "chk-2"),
	} {
		if other == base {
			t.Fatalf("distinct observations shared request key %q", base)
		}
	}
}

// TestCheckNarrative_SaysWhenTheResultChangedUnderTheCheck keeps the reason
// visible in durable evidence rather than only in the verdict.
func TestCheckNarrative_SaysWhenTheResultChangedUnderTheCheck(t *testing.T) {
	observation := passingObservation()
	observation.ArtifactVersion = "v1"
	summary, detail := checkNarrative(observation, true, "v2")
	if !strings.Contains(summary, "changed while checking") {
		t.Fatalf("summary = %q", summary)
	}
	if !strings.Contains(detail, "artifactChanged=v1->v2") {
		t.Fatalf("detail = %q", detail)
	}
	if !strings.Contains(detail, "enforcedBy=macos-seatbelt-workspace-write") {
		t.Fatalf("detail lost the enforcing mechanism: %q", detail)
	}
}
