package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

var requestNow = time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

func openRequest(token string) domain.DecompositionRequest {
	return domain.DecompositionRequest{
		ID: "dreq-1", OutcomeID: "out-parent", ContractRevisionID: "cr-1",
		Status:              domain.DecompositionRequested,
		CallbackTokenDigest: domain.HashCallbackToken(token),
		ExpiresAt:           requestNow.Add(10 * time.Minute),
		CreatedAt:           requestNow,
	}
}

func TestDecompositionRequestValidates(t *testing.T) {
	if err := openRequest("tok-abc").Validate(); err != nil {
		t.Fatalf("a well-formed request must validate: %v", err)
	}
	noDigest := openRequest("tok-abc")
	noDigest.CallbackTokenDigest = "not-a-digest"
	if err := noDigest.Validate(); err == nil || !strings.Contains(err.Error(), "hexadecimal") {
		t.Fatalf("a malformed digest must be refused, got %v", err)
	}
	noExpiry := openRequest("tok-abc")
	noExpiry.ExpiresAt = time.Time{}
	if err := noExpiry.Validate(); err == nil || !strings.Contains(err.Error(), "expiry") {
		t.Fatalf("expiry must be durable, got %v", err)
	}
}

// The token is stored only as a digest, so the record never carries the value
// an agent was handed.
func TestCallbackTokenIsOnlyEverStoredAsADigest(t *testing.T) {
	request := openRequest("tok-secret")
	if strings.Contains(request.CallbackTokenDigest, "tok-secret") {
		t.Fatal("the raw token must never appear in the stored record")
	}
	if !request.CallbackTokenMatches("tok-secret") {
		t.Fatal("the issued token must address its own request")
	}
	if request.CallbackTokenMatches("tok-other") || request.CallbackTokenMatches("") {
		t.Fatal("another token, or none, must not address this request")
	}
}

// Answering a closed, expired, or wrongly-addressed request is a ROUTING
// problem, not a validation one — it is refused before the proposal is parsed.
func TestAdmitProposalAnswerRefusesBeforeParsing(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*domain.DecompositionRequest)
		token   string
		now     time.Time
		wantHas string
	}{
		{name: "wrong token", token: "tok-other", now: requestNow, wantHas: "does not address"},
		{name: "no token", token: "", now: requestNow, wantHas: "does not address"},
		{
			name:   "already fulfilled",
			mutate: func(r *domain.DecompositionRequest) { r.Status = domain.DecompositionFulfilled },
			token:  "tok-abc", now: requestNow, wantHas: "already fulfilled",
		},
		{
			name:   "already rejected",
			mutate: func(r *domain.DecompositionRequest) { r.Status = domain.DecompositionRejected },
			token:  "tok-abc", now: requestNow, wantHas: "already rejected",
		},
		{name: "expired", token: "tok-abc", now: requestNow.Add(11 * time.Minute), wantHas: "expired"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := openRequest("tok-abc")
			if tt.mutate != nil {
				tt.mutate(&request)
			}
			err := request.AdmitProposalAnswer(tt.token, tt.now)
			if err == nil {
				t.Fatal("answer must be refused")
			}
			if !strings.Contains(err.Error(), tt.wantHas) {
				t.Fatalf("refusal %q must mention %q", err.Error(), tt.wantHas)
			}
		})
	}

	if err := openRequest("tok-abc").AdmitProposalAnswer("tok-abc", requestNow); err != nil {
		t.Fatalf("the issued token on an open, unexpired request must be admitted: %v", err)
	}
}

// Single-use is the point: the same token cannot answer twice, because the
// first answer closes the request.
func TestASecondAnswerIsRefused(t *testing.T) {
	request := openRequest("tok-abc")
	if err := request.AdmitProposalAnswer("tok-abc", requestNow); err != nil {
		t.Fatalf("first answer: %v", err)
	}
	request.Status = domain.DecompositionFulfilled // what the store does on success
	if err := request.AdmitProposalAnswer("tok-abc", requestNow); err == nil {
		t.Fatal("the same token must not answer a second time")
	}
}

// Expiry is derived from the clock, not from a state something has to advance,
// so a daemon that was not running still sees the truth on restart.
func TestExpiryIsDerivedNotStored(t *testing.T) {
	request := openRequest("tok-abc")
	if request.Expired(requestNow) {
		t.Fatal("a fresh request is not expired")
	}
	if !request.Expired(requestNow.Add(10 * time.Minute)) {
		t.Fatal("a request at its expiry is expired")
	}
	// A closed request is not "expired" — it already has a verdict.
	request.Status = domain.DecompositionFulfilled
	if request.Expired(requestNow.Add(time.Hour)) {
		t.Fatal("a fulfilled request must keep its verdict rather than becoming expired")
	}
}

func TestProposalSanityCapIsModest(t *testing.T) {
	if domain.MaxProposedContributions < 2 || domain.MaxProposedContributions > 20 {
		t.Fatalf("cap = %d; it bounds a runaway generation, not the product",
			domain.MaxProposedContributions)
	}
}
