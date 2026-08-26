package domain

import (
	"testing"
	"time"
)

var waldoConversationTestTime = time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)

func TestWaldoConversationAppendTurnValidatesProjectBindingAndOrderedSequence(t *testing.T) {
	conversation := WaldoConversation{
		ID: "waldo-conversation-1", ProjectID: "project-1", Revision: 3,
		LatestTurnSequence: 7, CreatedAt: waldoConversationTestTime, UpdatedAt: waldoConversationTestTime,
	}
	turn := WaldoConversationTurn{
		ID: "waldo-turn-8", ConversationID: conversation.ID, EpisodeID: "waldo-episode-1",
		ProjectID: conversation.ProjectID, Sequence: 8, Role: WaldoTurnRoleUser,
		Message:   "Keep the current contract revision and explain the next safe action.",
		CreatedAt: waldoConversationTestTime,
	}
	if err := turn.ValidateFor(conversation); err != nil {
		t.Fatalf("ValidateFor() error = %v", err)
	}

	wrongProject := turn
	wrongProject.ProjectID = "project-2"
	if err := wrongProject.ValidateFor(conversation); err == nil {
		t.Fatal("turn bound to another Project must fail")
	}

	outOfOrder := turn
	outOfOrder.Sequence = 10
	if err := outOfOrder.ValidateFor(conversation); err == nil {
		t.Fatal("turn that skips the next conversation sequence must fail")
	}
}

func TestWaldoContextRefRequiresTypedRevisionAndProvenance(t *testing.T) {
	valid := WaldoContextRef{
		Kind: WaldoContextOutcome, ObjectID: "outcome-1", Revision: "4",
		Provenance: WaldoContextProvenance{Kind: WaldoProvenanceCanonical, SourceID: "contract-revision-4"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	for name, mutate := range map[string]func(*WaldoContextRef){
		"untyped":          func(ref *WaldoContextRef) { ref.Kind = "free_form" },
		"missing revision": func(ref *WaldoContextRef) { ref.Revision = "" },
		"missing source":   func(ref *WaldoContextRef) { ref.Provenance.SourceID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatalf("Validate() accepted %+v", candidate)
			}
		})
	}

	project := WaldoContextRef{
		Kind: WaldoContextProject, ObjectID: "project-1",
		Provenance: WaldoContextProvenance{Kind: WaldoProvenanceUser, SourceID: "turn-1"},
	}
	if err := project.Validate(); err != nil {
		t.Fatalf("Project context does not need a synthetic revision: %v", err)
	}
}

func TestWaldoProviderTurnRefContainsOnlyOpaqueProviderReferences(t *testing.T) {
	ref := WaldoProviderTurnRef{
		Provider: "codex", ProviderConversationID: "thread-1", ProviderTurnID: "provider-turn-9",
		TranscriptRef: "native-transcript://session-1#turn-9",
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := ref.CanonicalKey(); got != "codex\x00thread-1\x00provider-turn-9\x00native-transcript://session-1#turn-9" {
		t.Fatalf("CanonicalKey() = %q", got)
	}
}

func TestAutomaticContinuationReceiptRequiresUnchangedBindingsAndSafeFacts(t *testing.T) {
	bindings := ContinuationBindings{
		ProjectID: "project-1", OutcomeID: "outcome-1", ContractRevisionID: "contract-1",
		PlanRevisionID: "plan-1", WorkUnitID: "work-unit-1", AttemptID: "attempt-1",
		Provider: "codex", Model: "gpt-5.6", Profile: "default", Role: "implementer",
		AuthorityDigest: digestOf('a'), BudgetDigest: digestOf('b'), WorkspaceOwner: "worktree-1",
		EffectPolicyDigest: digestOf('c'),
	}
	receipt := ContinuationReceipt{
		ID: "continuation-1", ConversationID: "waldo-conversation-1", ProjectID: "project-1",
		FromEpisodeID: "episode-1", ToEpisodeID: "episode-2",
		FromAgentSessionRef: "session-ref-1", ToAgentSessionRef: "session-ref-2",
		Action: ContinuationAutomatic, Reason: ContinuationReasonContextReserve,
		ReasonDetail:   "Provider reported insufficient trustworthy reserve.",
		MaterialChange: false, ContextDigest: digestOf('d'), PreviousBindings: bindings,
		ReplacementBindings: bindings, EffectsKnown: true, OldSessionFenced: true,
		ReplacementIdentityConfirmed: true, FenceReceiptRef: "fence-1",
		ReconciliationRef: "reconcile-1", CreatedAt: waldoConversationTestTime,
	}
	if err := receipt.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	for name, mutate := range map[string]func(*ContinuationReceipt){
		"unknown effect":      func(value *ContinuationReceipt) { value.EffectsKnown = false },
		"unsafe fence":        func(value *ContinuationReceipt) { value.OldSessionFenced = false },
		"ambiguous identity":  func(value *ContinuationReceipt) { value.ReplacementIdentityConfirmed = false },
		"changed provider":    func(value *ContinuationReceipt) { value.ReplacementBindings.Provider = "deepseek-harness" },
		"missing replacement": func(value *ContinuationReceipt) { value.ToAgentSessionRef = "" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := receipt
			mutate(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatalf("Validate() accepted unsafe automatic continuation: %+v", candidate)
			}
		})
	}
}

func TestNeedsYouContinuationReceiptPreservesMaterialChangeWithoutReplacement(t *testing.T) {
	before := ContinuationBindings{
		ProjectID: "project-1", OutcomeID: "outcome-1", ContractRevisionID: "contract-1",
		PlanRevisionID: "plan-1", WorkUnitID: "work-unit-1", AttemptID: "attempt-1",
		Provider: "codex", Model: "gpt-5.6", Profile: "default", Role: "implementer",
		AuthorityDigest: digestOf('a'), BudgetDigest: digestOf('b'), WorkspaceOwner: "worktree-1",
		EffectPolicyDigest: digestOf('c'),
	}
	after := before
	after.Role = "verifier"
	receipt := ContinuationReceipt{
		ID: "continuation-needs-you", ConversationID: "waldo-conversation-1", ProjectID: "project-1",
		FromEpisodeID:       "episode-1",
		FromAgentSessionRef: "session-ref-1", Action: ContinuationNeedsYou,
		Reason: ContinuationReasonFreshVerifier, ReasonDetail: "Verification requires an independent context.",
		MaterialChange: true, ChangedFields: []string{"role"}, ContextDigest: digestOf('d'),
		PreviousBindings: before, ReplacementBindings: after, EffectsKnown: true,
		NeedsUserReason: "Start a fresh verifier Attempt without implementer conclusions.",
		CreatedAt:       waldoConversationTestTime,
	}
	if err := receipt.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !receipt.ToAgentSessionRef.IsZero() {
		t.Fatal("Needs You receipt must not invent a replacement session identity")
	}
}

func digestOf(value byte) string {
	buf := make([]byte, 64)
	for index := range buf {
		buf[index] = value
	}
	return string(buf)
}
