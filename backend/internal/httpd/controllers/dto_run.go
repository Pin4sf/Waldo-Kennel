package controllers

import "time"

// DeliveryIDParam is the {deliveryId} path parameter on delivery reads.
type DeliveryIDParam struct {
	DeliveryID string `path:"deliveryId" description:"Delivery identifier, e.g. dlv-<uuid>."`
}

// OutcomeRunScopeParam widens a Board listing to include contributing
// Outcomes, which are otherwise shown inside their parent's Mission.
type OutcomeRunScopeParam struct {
	Scope string `query:"scope" description:"top_level (default) lists only Project-level Outcomes; all includes contributing Outcomes." enum:"top_level,all"`
}

// RunActionEligibilityResponse is one owner action and whether the daemon
// will honour it right now. Reason is a stable policy code, present exactly
// when the action is unavailable; the renderer owns the wording.
type RunActionEligibilityResponse struct {
	Action    string `json:"action" enum:"clarify,propose_plan,review_plan,approve_plan,start,pause,resume,cancel,review_result,request_changes,accept,export"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// RunIntentResponse is the persisted authorization to keep running a Plan.
// It is absent until durable run intent exists; absent means only per-Attempt
// Start is available, not that a run is idle.
type RunIntentResponse struct {
	Generation     int64  `json:"generation"`
	Desired        string `json:"desired" enum:"idle,running,paused,cancelled"`
	PlanRevisionID string `json:"planRevisionId,omitempty"`
	// BindsCurrentPlan is false when the authorization names a Plan revision
	// the Outcome has moved past. Continuation schedules against the Plan the
	// intent names, so a superseded one can admit nothing until it is cancelled.
	BindsCurrentPlan bool      `json:"bindsCurrentPlan"`
	RequestedAt      time.Time `json:"requestedAt"`
	// AcknowledgedAt is what separates an acknowledged pause/cancel from a
	// requested one. Absent means the request has not yet taken effect.
	AcknowledgedAt  *time.Time `json:"acknowledgedAt,omitempty"`
	ActiveAttemptID string     `json:"activeAttemptId,omitempty"`
	LastError       string     `json:"lastError,omitempty"`
}

// RunBlockerResponse is the single concrete reason an Outcome needs its owner.
type RunBlockerResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail,omitempty"`
}

// RunFreshnessResponse lets a client discard a response older than state it
// already holds, which is what stops a slow reply overwriting a newer screen.
type RunFreshnessResponse struct {
	ObservedAt             time.Time `json:"observedAt"`
	ContractRevisionNumber int64     `json:"contractRevisionNumber"`
	PlanRevisionID         string    `json:"planRevisionId,omitempty"`
	// ProofGeneration is the append-only proof record count this projection
	// was computed from. A response carrying a lower value than one already
	// held is stale.
	ProofGeneration int64 `json:"proofGeneration"`
}

// OutcomeRunStateResponse is the Mission header projection: milestone, the
// owner's next moves, and what is in the way. Every field is derived from
// canonical Contract/Plan/Attempt/proof facts on read and none is stored.
type OutcomeRunStateResponse struct {
	OutcomeID       string `json:"outcomeId"`
	ProjectID       string `json:"projectId"`
	Title           string `json:"title"`
	ParentOutcomeID string `json:"parentOutcomeId,omitempty"`
	State           string `json:"state" enum:"define,ready_to_authorize,in_progress,needs_you,ready_for_review,accepted"`
	// AttentionReason is a stable code, non-empty exactly when state is
	// needs_you, so "Needs you" always carries a concrete reason.
	AttentionReason          string                         `json:"attentionReason,omitempty"`
	Intent                   *RunIntentResponse             `json:"intent,omitempty"`
	EligibleActions          []RunActionEligibilityResponse `json:"eligibleActions"`
	Blocker                  *RunBlockerResponse            `json:"blocker,omitempty"`
	Freshness                RunFreshnessResponse           `json:"freshness"`
	PlanStatus               string                         `json:"planStatus,omitempty"`
	PlanBindsCurrentContract bool                           `json:"planBindsCurrentContract"`
	ActiveAttemptID          string                         `json:"activeAttemptId,omitempty"`
	ActiveAttemptStatus      string                         `json:"activeAttemptStatus,omitempty"`
	// ProvenCriteria and RequiredCriteria are counts, deliberately not a
	// percentage: Kennel does not know how much work remains.
	ProvenCriteria   int        `json:"provenCriteria"`
	RequiredCriteria int        `json:"requiredCriteria"`
	AcceptedAt       *time.Time `json:"acceptedAt,omitempty"`
}

// OutcomeRunStateEnvelope is the { runState } body for one Outcome.
type OutcomeRunStateEnvelope struct {
	RunState OutcomeRunStateResponse `json:"runState"`
}

// OutcomeRunStatesEnvelope is the Board projection for one Project: every
// top-level Outcome's Mission state in one round trip.
type OutcomeRunStatesEnvelope struct {
	RunStates  []OutcomeRunStateResponse `json:"runStates"`
	ObservedAt time.Time                 `json:"observedAt"`
}

// OutcomeRunCommandRequest is the owner's durable run intent command.
// RequestKey is the replay identity: repeating a key returns the same state
// rather than acting twice, so double-click and reconnect retry are safe.
type OutcomeRunCommandRequest struct {
	Action                   string `json:"action" enum:"start,pause,resume,cancel"`
	PlanRevisionID           string `json:"planRevisionId"`
	ExpectedContractRevision int64  `json:"expectedContractRevision"`
	// ExpectedGeneration is optimistic concurrency against the intent the
	// caller last read. Zero means "no expectation".
	ExpectedGeneration int64  `json:"expectedGeneration,omitempty"`
	RequestKey         string `json:"requestKey"`
}

// OutcomeDeliveryResponse is one durable delivery of one exact artifact.
// Delivery is transfer, never merge, publication or acceptance.
type OutcomeDeliveryResponse struct {
	ID              string `json:"id"`
	OutcomeID       string `json:"outcomeId"`
	AttemptID       string `json:"attemptId"`
	WorkUnitID      string `json:"workUnitId,omitempty"`
	ArtifactVersion string `json:"artifactVersion"`
	Disposition     string `json:"disposition" enum:"accepted,draft"`
	Destination     string `json:"destination"`
	State           string `json:"state" enum:"pending,succeeded,failed,cancelled"`
	// CompletionSource says how a terminal result was established. "observed"
	// means the daemon watched the transfer resolve; "recovered" means the row
	// was left pending by a crash between the filesystem commit and the ledger
	// write, and the result was established afterwards by reading the
	// destination's manifest and re-digesting every delivered artifact against
	// the retained result. A recovered success proves the bytes arrived; nobody
	// watched them arrive. Absent while pending.
	CompletionSource string `json:"completionSource,omitempty" enum:"observed,recovered"`
	ManifestPath     string `json:"manifestPath,omitempty"`
	FileCount        int    `json:"fileCount"`
	ByteCount        int64  `json:"byteCount"`
	FailureCode      string `json:"failureCode,omitempty"`
	FailureDetail    string `json:"failureDetail,omitempty"`

	RequestedAt time.Time  `json:"requestedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// OutcomeDeliveryEnvelope is the { delivery } body for one delivery.
type OutcomeDeliveryEnvelope struct {
	Delivery OutcomeDeliveryResponse `json:"delivery"`
}

// OutcomeDeliveriesEnvelope is the { deliveries } collection body.
type OutcomeDeliveriesEnvelope struct {
	Deliveries []OutcomeDeliveryResponse `json:"deliveries"`
}

// RequestOutcomeDeliveryRequest asks for delivery of one exact reviewed
// artifact. The artifact version is required so a later Attempt's output can
// never be delivered as the one the owner reviewed.
type RequestOutcomeDeliveryRequest struct {
	AttemptID            string `json:"attemptId"`
	ArtifactVersion      string `json:"artifactVersion"`
	Destination          string `json:"destination"`
	Disposition          string `json:"disposition" enum:"accepted,draft"`
	AcceptanceDecisionID string `json:"acceptanceDecisionId,omitempty"`
	RequestKey           string `json:"requestKey"`
}

// AttributedUsageResponse is one attributed usage block. It reuses the
// normalized telemetry totals every other usage surface reports, where every
// numeric is nullable and null means unknown — never zero. CostEstimated stays
// true unless PricingProvenance names a real price source, because a cost
// without provenance is a guess and must not be printed as a fact.
type AttributedUsageResponse struct {
	Totals            UsageTotalsResponse `json:"totals"`
	Requests          *int64              `json:"requests"`
	CostUsd           *float64            `json:"costUsd"`
	CostEstimated     bool                `json:"costEstimated"`
	PricingProvenance string              `json:"pricingProvenance,omitempty"`
}

// OutcomeUsageWorkUnitResponse attributes usage to one canonical WorkUnit.
type OutcomeUsageWorkUnitResponse struct {
	WorkUnitID string                  `json:"workUnitId"`
	Usage      AttributedUsageResponse `json:"usage"`
}

// OutcomeUsageAttemptResponse attributes usage to one Attempt.
type OutcomeUsageAttemptResponse struct {
	AttemptID  string                  `json:"attemptId"`
	WorkUnitID string                  `json:"workUnitId"`
	Usage      AttributedUsageResponse `json:"usage"`
}

// OutcomeUsageEnvelope is the attributed-usage body for one Outcome.
// Planning usage is reported separately from execution so reasoning spend is
// never presented as work the provider did.
type OutcomeUsageEnvelope struct {
	OutcomeID  string                         `json:"outcomeId"`
	Planning   AttributedUsageResponse        `json:"planning"`
	Totals     AttributedUsageResponse        `json:"totals"`
	WorkUnits  []OutcomeUsageWorkUnitResponse `json:"workUnits"`
	Attempts   []OutcomeUsageAttemptResponse  `json:"attempts"`
	ObservedAt time.Time                      `json:"observedAt"`
}

// SelectOutcomeDocumentsRequest is the owner's explicit selection of local
// documents. Paths are absolute and chosen by the owner; nothing is
// discovered by traversal.
type SelectOutcomeDocumentsRequest struct {
	Paths []string `json:"paths"`
}

// ApproveOutcomeDocumentsRequest confirms the reviewed scope. The digest is
// required so a selection made after the owner looked away cannot be approved
// by a click aimed at the one they read.
type ApproveOutcomeDocumentsRequest struct {
	ExpectedDigest string `json:"expectedDigest"`
}

// DocumentSourceResponse is one selected document as it was when selected.
// SourcePath is provenance: execution reads the snapshot, never the original.
type DocumentSourceResponse struct {
	ID            string `json:"id"`
	Position      int    `json:"position"`
	SourcePath    string `json:"sourcePath"`
	Name          string `json:"name"`
	ContentDigest string `json:"contentDigest"`
	SizeBytes     int64  `json:"sizeBytes"`
}

// OutcomeDocumentContextResponse is one immutable selection revision.
type OutcomeDocumentContextResponse struct {
	ID        string `json:"id"`
	OutcomeID string `json:"outcomeId"`
	Revision  int64  `json:"revision"`
	// Digest identifies the selected bytes as a set and is what binds a
	// proposal to the material it was grounded in.
	Digest     string                   `json:"digest"`
	State      string                   `json:"state" enum:"selected,approved"`
	SelectedAt time.Time                `json:"selectedAt"`
	ApprovedAt *time.Time               `json:"approvedAt,omitempty"`
	Sources    []DocumentSourceResponse `json:"sources"`
	// ChangedSources names documents whose bytes on disk no longer match the
	// approved snapshot. The run still uses the snapshot; this is what lets
	// the owner deliberately re-select instead of finding out in the result.
	ChangedSources []string `json:"changedSources"`
}

// OutcomeDocumentContextEnvelope is the { documentContext } body.
type OutcomeDocumentContextEnvelope struct {
	DocumentContext OutcomeDocumentContextResponse `json:"documentContext"`
}
