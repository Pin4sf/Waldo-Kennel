package domain

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// WaldoConversationID identifies the single durable Waldo conversation for a Project.
type WaldoConversationID string

// IsZero reports whether the conversation id is empty.
func (id WaldoConversationID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

func (id WaldoConversationID) String() string { return string(id) }

// WaldoConversationEpisodeID identifies one bounded provider-neutral episode.
type WaldoConversationEpisodeID string

// IsZero reports whether the episode id is empty.
func (id WaldoConversationEpisodeID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

func (id WaldoConversationEpisodeID) String() string { return string(id) }

// WaldoConversationTurnID identifies one ordered visible Project-Waldo turn.
type WaldoConversationTurnID string

// IsZero reports whether the turn id is empty.
func (id WaldoConversationTurnID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

func (id WaldoConversationTurnID) String() string { return string(id) }

// WaldoContextAttachmentID identifies one explicit context attachment lifecycle.
type WaldoContextAttachmentID string

// IsZero reports whether the context attachment id is empty.
func (id WaldoContextAttachmentID) IsZero() bool { return strings.TrimSpace(string(id)) == "" }

func (id WaldoContextAttachmentID) String() string { return string(id) }

// WaldoConversation is the durable Project-scoped aggregate root. Revision is
// optimistic concurrency truth; LatestTurnSequence is the sole ordered-turn allocator.
type WaldoConversation struct {
	ID                 WaldoConversationID
	ProjectID          ProjectID
	Revision           int64
	LatestTurnSequence int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Validate enforces aggregate identity, revision, and timestamp invariants.
func (conversation WaldoConversation) Validate() error {
	if conversation.ID.IsZero() {
		return fmt.Errorf("waldo conversation id is required")
	}
	if strings.TrimSpace(string(conversation.ProjectID)) == "" {
		return fmt.Errorf("waldo conversation Project id is required")
	}
	if conversation.Revision < 0 || conversation.LatestTurnSequence < 0 {
		return fmt.Errorf("waldo conversation revision and sequence must not be negative")
	}
	if conversation.CreatedAt.IsZero() || conversation.UpdatedAt.IsZero() {
		return fmt.Errorf("waldo conversation timestamps are required")
	}
	return nil
}

// WaldoEpisodeState records whether an episode can receive more turns.
type WaldoEpisodeState string

// Waldo episode states distinguish the current writable episode from history.
const (
	WaldoEpisodeActive WaldoEpisodeState = "active"
	WaldoEpisodeSealed WaldoEpisodeState = "sealed"
)

// Valid reports whether state is a supported persisted episode state.
func (state WaldoEpisodeState) Valid() bool {
	return state == WaldoEpisodeActive || state == WaldoEpisodeSealed
}

// WaldoProviderEpisodeRef identifies provider-native history without copying it.
type WaldoProviderEpisodeRef struct {
	Provider               AgentHarness
	ProviderConversationID string
	TranscriptRef          string
}

// Validate enforces an identifier-only provider episode reference.
func (ref WaldoProviderEpisodeRef) Validate() error {
	if strings.TrimSpace(string(ref.Provider)) == "" {
		return fmt.Errorf("provider episode provider is required")
	}
	if strings.TrimSpace(ref.ProviderConversationID) == "" && strings.TrimSpace(ref.TranscriptRef) == "" {
		return fmt.Errorf("provider episode requires a conversation or transcript reference")
	}
	return nil
}

// WaldoConversationEpisode is one bounded execution episode beneath the
// persistent Project relationship. It carries opaque provider references only.
type WaldoConversationEpisode struct {
	ID             WaldoConversationEpisodeID
	ConversationID WaldoConversationID
	ProjectID      ProjectID
	Ordinal        int64
	State          WaldoEpisodeState
	ProviderRef    *WaldoProviderEpisodeRef
	CreatedAt      time.Time
	SealedAt       *time.Time
	SealReason     string
}

// Validate enforces episode binding and one-way sealing facts.
func (episode WaldoConversationEpisode) Validate() error {
	if episode.ID.IsZero() || episode.ConversationID.IsZero() {
		return fmt.Errorf("waldo episode identity is required")
	}
	if strings.TrimSpace(string(episode.ProjectID)) == "" {
		return fmt.Errorf("waldo episode Project id is required")
	}
	if episode.Ordinal < 1 {
		return fmt.Errorf("waldo episode ordinal must be at least 1")
	}
	if !episode.State.Valid() {
		return fmt.Errorf("unsupported Waldo episode state %q", episode.State)
	}
	if episode.CreatedAt.IsZero() {
		return fmt.Errorf("waldo episode created time is required")
	}
	if episode.ProviderRef != nil {
		if err := episode.ProviderRef.Validate(); err != nil {
			return err
		}
	}
	if episode.State == WaldoEpisodeActive && (episode.SealedAt != nil || strings.TrimSpace(episode.SealReason) != "") {
		return fmt.Errorf("active Waldo episode cannot carry seal facts")
	}
	if episode.State == WaldoEpisodeSealed && (episode.SealedAt == nil || strings.TrimSpace(episode.SealReason) == "") {
		return fmt.Errorf("sealed Waldo episode requires time and reason")
	}
	return nil
}

// WaldoTurnRole distinguishes the owner's text from Waldo's visible response.
type WaldoTurnRole string

// Waldo turn roles distinguish owner intent from Waldo-visible responses.
const (
	WaldoTurnRoleUser  WaldoTurnRole = "user"
	WaldoTurnRoleWaldo WaldoTurnRole = "waldo"
)

// Valid reports whether role is a supported visible turn role.
func (role WaldoTurnRole) Valid() bool {
	return role == WaldoTurnRoleUser || role == WaldoTurnRoleWaldo
}

// WaldoProviderTurnRef identifies the provider-native source for a visible
// response. It intentionally has no transcript content or message body.
type WaldoProviderTurnRef struct {
	Provider               AgentHarness
	ProviderConversationID string
	ProviderTurnID         string
	TranscriptRef          string
}

// Validate enforces an identifier-only provider turn reference.
func (ref WaldoProviderTurnRef) Validate() error {
	if strings.TrimSpace(string(ref.Provider)) == "" {
		return fmt.Errorf("provider turn provider is required")
	}
	if strings.TrimSpace(ref.ProviderTurnID) == "" {
		return fmt.Errorf("provider turn id is required")
	}
	if strings.TrimSpace(ref.ProviderConversationID) == "" && strings.TrimSpace(ref.TranscriptRef) == "" {
		return fmt.Errorf("provider turn requires a conversation or transcript reference")
	}
	return nil
}

// CanonicalKey returns a stable identifier-only fingerprint input.
func (ref WaldoProviderTurnRef) CanonicalKey() string {
	return strings.Join([]string{string(ref.Provider), ref.ProviderConversationID, ref.ProviderTurnID, ref.TranscriptRef}, "\x00")
}

// WaldoContextRefKind names canonical objects that may be attached explicitly.
type WaldoContextRefKind string

// Waldo context kinds enumerate canonical Project-bound objects.
const (
	WaldoContextProject          WaldoContextRefKind = "project"
	WaldoContextOutcome          WaldoContextRefKind = "outcome"
	WaldoContextContractRevision WaldoContextRefKind = "contract_revision"
	WaldoContextPlanRevision     WaldoContextRefKind = "plan_revision"
	WaldoContextWorkUnit         WaldoContextRefKind = "work_unit"
	WaldoContextAttempt          WaldoContextRefKind = "attempt"
	WaldoContextAgentSessionRef  WaldoContextRefKind = "agent_session_ref"
	WaldoContextIntakeSession    WaldoContextRefKind = "intake_session"
)

// Valid reports whether kind is a supported canonical context type.
func (kind WaldoContextRefKind) Valid() bool {
	switch kind {
	case WaldoContextProject, WaldoContextOutcome, WaldoContextContractRevision,
		WaldoContextPlanRevision, WaldoContextWorkUnit, WaldoContextAttempt,
		WaldoContextAgentSessionRef, WaldoContextIntakeSession:
		return true
	default:
		return false
	}
}

// WaldoContextProvenanceKind identifies why a context object is eligible.
type WaldoContextProvenanceKind string

// Waldo provenance kinds record why a context object was eligible.
const (
	WaldoProvenanceUser       WaldoContextProvenanceKind = "user"
	WaldoProvenanceCanonical  WaldoContextProvenanceKind = "canonical"
	WaldoProvenanceIntake     WaldoContextProvenanceKind = "intake"
	WaldoProvenanceProvider   WaldoContextProvenanceKind = "provider"
	WaldoProvenanceRetrieval  WaldoContextProvenanceKind = "retrieval"
	WaldoProvenanceCorrection WaldoContextProvenanceKind = "correction"
)

// Valid reports whether kind is a supported provenance type.
func (kind WaldoContextProvenanceKind) Valid() bool {
	switch kind {
	case WaldoProvenanceUser, WaldoProvenanceCanonical, WaldoProvenanceIntake,
		WaldoProvenanceProvider, WaldoProvenanceRetrieval, WaldoProvenanceCorrection:
		return true
	default:
		return false
	}
}

// WaldoContextProvenance preserves the exact source of an attachment.
type WaldoContextProvenance struct {
	Kind     WaldoContextProvenanceKind
	SourceID string
}

// Validate enforces typed provenance with an exact source identifier.
func (provenance WaldoContextProvenance) Validate() error {
	if !provenance.Kind.Valid() {
		return fmt.Errorf("unsupported Waldo context provenance %q", provenance.Kind)
	}
	if strings.TrimSpace(provenance.SourceID) == "" {
		return fmt.Errorf("waldo context provenance source id is required")
	}
	return nil
}

// WaldoContextRef is an identifier-only canonical context reference. Revision
// is required for every mutable or revisioned object; Project identity itself
// is the only unrevisioned reference.
type WaldoContextRef struct {
	Kind       WaldoContextRefKind
	ObjectID   string
	Revision   string
	Provenance WaldoContextProvenance
}

// Validate enforces a typed, provenance-bearing canonical reference.
func (ref WaldoContextRef) Validate() error {
	if !ref.Kind.Valid() {
		return fmt.Errorf("unsupported Waldo context kind %q", ref.Kind)
	}
	if strings.TrimSpace(ref.ObjectID) == "" {
		return fmt.Errorf("waldo context object id is required")
	}
	if ref.Kind != WaldoContextProject && strings.TrimSpace(ref.Revision) == "" {
		return fmt.Errorf("waldo context revision is required for %s", ref.Kind)
	}
	return ref.Provenance.Validate()
}

// WaldoContextAttachment is the explicit attach/detach lifecycle for one ref.
type WaldoContextAttachment struct {
	ID               WaldoContextAttachmentID
	ConversationID   WaldoConversationID
	ProjectID        ProjectID
	Ref              WaldoContextRef
	AttachedRevision int64
	DetachedRevision int64
	CreatedAt        time.Time
	DetachedAt       *time.Time
	DetachReason     string
}

// Active reports whether the attachment remains eligible for future turns.
func (attachment WaldoContextAttachment) Active() bool { return attachment.DetachedAt == nil }

// Validate enforces explicit attach and one-time detach facts.
func (attachment WaldoContextAttachment) Validate() error {
	if attachment.ID.IsZero() || attachment.ConversationID.IsZero() {
		return fmt.Errorf("waldo context attachment identity is required")
	}
	if strings.TrimSpace(string(attachment.ProjectID)) == "" {
		return fmt.Errorf("waldo context attachment Project id is required")
	}
	if err := attachment.Ref.Validate(); err != nil {
		return err
	}
	if attachment.AttachedRevision < 1 {
		return fmt.Errorf("waldo context attached revision must be at least 1")
	}
	if attachment.CreatedAt.IsZero() {
		return fmt.Errorf("waldo context created time is required")
	}
	if attachment.Active() && (attachment.DetachedRevision != 0 || strings.TrimSpace(attachment.DetachReason) != "") {
		return fmt.Errorf("active Waldo context cannot carry detach facts")
	}
	if !attachment.Active() && (attachment.DetachedRevision <= attachment.AttachedRevision || strings.TrimSpace(attachment.DetachReason) == "") {
		return fmt.Errorf("detached Waldo context requires a later revision and reason")
	}
	return nil
}

// WaldoConversationTurn is one ordered user or Waldo-visible message. The
// visible message is not a provider transcript: provider-native history stays
// behind the opaque ProviderRef.
type WaldoConversationTurn struct {
	ID             WaldoConversationTurnID
	ConversationID WaldoConversationID
	EpisodeID      WaldoConversationEpisodeID
	ProjectID      ProjectID
	Sequence       int64
	Role           WaldoTurnRole
	Message        string
	ProviderRef    *WaldoProviderTurnRef
	ContextRefs    []WaldoContextRef
	CreatedAt      time.Time
}

// ValidateFor enforces Project binding and the next aggregate turn sequence.
func (turn WaldoConversationTurn) ValidateFor(conversation WaldoConversation) error {
	if err := conversation.Validate(); err != nil {
		return err
	}
	if turn.ID.IsZero() || turn.ConversationID.IsZero() || turn.EpisodeID.IsZero() {
		return fmt.Errorf("waldo conversation turn identity is required")
	}
	if turn.ConversationID != conversation.ID {
		return fmt.Errorf("waldo turn binds another conversation")
	}
	if turn.ProjectID != conversation.ProjectID {
		return fmt.Errorf("waldo turn binds another Project")
	}
	if turn.Sequence != conversation.LatestTurnSequence+1 {
		return fmt.Errorf("waldo turn sequence %d is not next after %d", turn.Sequence, conversation.LatestTurnSequence)
	}
	if !turn.Role.Valid() {
		return fmt.Errorf("unsupported Waldo turn role %q", turn.Role)
	}
	if strings.TrimSpace(turn.Message) == "" {
		return fmt.Errorf("waldo turn message is required")
	}
	if turn.ProviderRef != nil {
		if err := turn.ProviderRef.Validate(); err != nil {
			return err
		}
	}
	for index, ref := range turn.ContextRefs {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("waldo turn context %d: %w", index+1, err)
		}
	}
	if turn.CreatedAt.IsZero() {
		return fmt.Errorf("waldo turn created time is required")
	}
	return nil
}

// ContinuationBindings freezes every material dimension used by rollover policy.
type ContinuationBindings struct {
	ProjectID          ProjectID
	OutcomeID          OutcomeID
	ContractRevisionID ContractRevisionID
	PlanRevisionID     PlanRevisionID
	WorkUnitID         WorkUnitID
	AttemptID          AttemptID
	Provider           AgentHarness
	Model              string
	Profile            string
	Role               string
	AuthorityDigest    string
	BudgetDigest       string
	WorkspaceOwner     string
	EffectPolicyDigest string
}

// Validate enforces the complete material binding tuple used by policy.
func (bindings ContinuationBindings) Validate() error {
	if strings.TrimSpace(string(bindings.ProjectID)) == "" || bindings.OutcomeID.IsZero() ||
		bindings.ContractRevisionID.IsZero() || bindings.PlanRevisionID.IsZero() ||
		bindings.WorkUnitID.IsZero() || bindings.AttemptID.IsZero() {
		return fmt.Errorf("continuation canonical bindings are incomplete")
	}
	if strings.TrimSpace(string(bindings.Provider)) == "" || strings.TrimSpace(bindings.Role) == "" ||
		strings.TrimSpace(bindings.WorkspaceOwner) == "" {
		return fmt.Errorf("continuation provider, role, and workspace owner are required")
	}
	for label, digest := range map[string]string{
		"authority":     bindings.AuthorityDigest,
		"budget":        bindings.BudgetDigest,
		"effect policy": bindings.EffectPolicyDigest,
	} {
		if !validDigest(digest) {
			return fmt.Errorf("continuation %s digest must be SHA-256 hex", label)
		}
	}
	return nil
}

// Equal reports whether a replacement preserves every automatic-continuation binding.
func (bindings ContinuationBindings) Equal(other ContinuationBindings) bool {
	return bindings == other
}

// Changed reports the exact material binding fields that differ.
func (bindings ContinuationBindings) Changed(other ContinuationBindings) []string {
	var changed []string
	checks := []struct {
		name  string
		equal bool
	}{
		{"project", bindings.ProjectID == other.ProjectID},
		{"outcome", bindings.OutcomeID == other.OutcomeID},
		{"contract_revision", bindings.ContractRevisionID == other.ContractRevisionID},
		{"plan_revision", bindings.PlanRevisionID == other.PlanRevisionID},
		{"work_unit", bindings.WorkUnitID == other.WorkUnitID},
		{"attempt", bindings.AttemptID == other.AttemptID},
		{"provider", bindings.Provider == other.Provider},
		{"model", bindings.Model == other.Model},
		{"profile", bindings.Profile == other.Profile},
		{"role", bindings.Role == other.Role},
		{"authority", bindings.AuthorityDigest == other.AuthorityDigest},
		{"budget", bindings.BudgetDigest == other.BudgetDigest},
		{"workspace", bindings.WorkspaceOwner == other.WorkspaceOwner},
		{"effect_policy", bindings.EffectPolicyDigest == other.EffectPolicyDigest},
	}
	for _, check := range checks {
		if !check.equal {
			changed = append(changed, check.name)
		}
	}
	return changed
}

// ContinuationReason records the exact trigger for a continuation decision.
type ContinuationReason string

// Continuation reasons preserve the exact bounded-rollover trigger.
const (
	ContinuationReasonContextReserve        ContinuationReason = "context_reserve"
	ContinuationReasonConservativeThreshold ContinuationReason = "conservative_threshold"
	ContinuationReasonMaterialDigestChange  ContinuationReason = "material_digest_change"
	ContinuationReasonIdentityLost          ContinuationReason = "identity_lost"
	ContinuationReasonSourceRevoked         ContinuationReason = "source_revoked"
	ContinuationReasonFreshVerifier         ContinuationReason = "fresh_verifier"
	ContinuationReasonUserRequested         ContinuationReason = "user_requested"
)

// Valid reports whether reason is a supported continuation trigger.
func (reason ContinuationReason) Valid() bool {
	switch reason {
	case ContinuationReasonContextReserve, ContinuationReasonConservativeThreshold,
		ContinuationReasonMaterialDigestChange, ContinuationReasonIdentityLost,
		ContinuationReasonSourceRevoked, ContinuationReasonFreshVerifier,
		ContinuationReasonUserRequested:
		return true
	default:
		return false
	}
}

// ContinuationAction is the durable policy decision, not a display status.
type ContinuationAction string

// Continuation actions distinguish safe automation from durable stops.
const (
	ContinuationAutomatic   ContinuationAction = "automatic"
	ContinuationNeedsYou    ContinuationAction = "needs_you"
	ContinuationUnconfirmed ContinuationAction = "unconfirmed"
)

// Valid reports whether action is a supported persisted policy decision.
func (action ContinuationAction) Valid() bool {
	return action == ContinuationAutomatic || action == ContinuationNeedsYou || action == ContinuationUnconfirmed
}

// ContinuationReceipt is the append-only lineage and policy receipt between
// an old AgentSessionRef and either a confirmed replacement or a durable stop.
type ContinuationReceipt struct {
	ID                           string
	ConversationID               WaldoConversationID
	ProjectID                    ProjectID
	FromEpisodeID                WaldoConversationEpisodeID
	ToEpisodeID                  WaldoConversationEpisodeID
	FromAgentSessionRef          AttemptSessionRefID
	ToAgentSessionRef            AttemptSessionRefID
	Action                       ContinuationAction
	Reason                       ContinuationReason
	ReasonDetail                 string
	MaterialChange               bool
	ChangedFields                []string
	ContextDigest                string
	ContextRefs                  []WaldoContextRef
	PreviousBindings             ContinuationBindings
	ReplacementBindings          ContinuationBindings
	EffectsKnown                 bool
	OldSessionFenced             bool
	ReplacementIdentityConfirmed bool
	FenceReceiptRef              string
	ReconciliationRef            string
	NeedsUserReason              string
	CreatedAt                    time.Time
}

// Validate enforces fail-closed continuation lineage and replacement truth.
func (receipt ContinuationReceipt) Validate() error {
	if strings.TrimSpace(receipt.ID) == "" || receipt.ConversationID.IsZero() || receipt.FromEpisodeID.IsZero() ||
		strings.TrimSpace(string(receipt.ProjectID)) == "" || receipt.FromAgentSessionRef.IsZero() {
		return fmt.Errorf("continuation receipt identity is required")
	}
	if !receipt.Action.Valid() || !receipt.Reason.Valid() {
		return fmt.Errorf("unsupported continuation action or reason")
	}
	if strings.TrimSpace(receipt.ReasonDetail) == "" || !validDigest(receipt.ContextDigest) {
		return fmt.Errorf("continuation reason detail and context digest are required")
	}
	for index, ref := range receipt.ContextRefs {
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("continuation context %d: %w", index+1, err)
		}
	}
	if err := receipt.PreviousBindings.Validate(); err != nil {
		return err
	}
	if err := receipt.ReplacementBindings.Validate(); err != nil {
		return err
	}
	if receipt.PreviousBindings.ProjectID != receipt.ProjectID {
		return fmt.Errorf("continuation predecessor bindings must match receipt Project")
	}
	if receipt.CreatedAt.IsZero() {
		return fmt.Errorf("continuation receipt created time is required")
	}
	switch receipt.Action {
	case ContinuationAutomatic:
		if receipt.MaterialChange || len(receipt.ChangedFields) != 0 || !receipt.PreviousBindings.Equal(receipt.ReplacementBindings) {
			return fmt.Errorf("automatic continuation cannot change material bindings")
		}
		if !receipt.EffectsKnown || !receipt.OldSessionFenced || !receipt.ReplacementIdentityConfirmed {
			return fmt.Errorf("automatic continuation requires known effects, safe fencing, and confirmed identity")
		}
		if receipt.ToEpisodeID.IsZero() || receipt.ToEpisodeID == receipt.FromEpisodeID ||
			receipt.ToAgentSessionRef.IsZero() || receipt.ToAgentSessionRef == receipt.FromAgentSessionRef {
			return fmt.Errorf("automatic continuation requires a distinct confirmed replacement session ref")
		}
		if strings.TrimSpace(receipt.FenceReceiptRef) == "" || strings.TrimSpace(receipt.ReconciliationRef) == "" {
			return fmt.Errorf("automatic continuation requires fencing and reconciliation facts")
		}
	case ContinuationNeedsYou:
		if !receipt.ToEpisodeID.IsZero() || !receipt.ToAgentSessionRef.IsZero() {
			return fmt.Errorf("needs you continuation cannot claim a replacement identity")
		}
		if strings.TrimSpace(receipt.NeedsUserReason) == "" {
			return fmt.Errorf("needs you continuation requires an exact owner decision reason")
		}
	case ContinuationUnconfirmed:
		if !receipt.ToEpisodeID.IsZero() || !receipt.ToAgentSessionRef.IsZero() || receipt.ReplacementIdentityConfirmed {
			return fmt.Errorf("unconfirmed continuation cannot claim a replacement identity")
		}
		if strings.TrimSpace(receipt.NeedsUserReason) == "" || !receipt.OldSessionFenced {
			return fmt.Errorf("unconfirmed continuation requires durable attention and a fenced old session")
		}
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
