package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AttemptCompletionBoundary names the provider fact that may end an approved
// WorkUnit's subordinate session. Empty is legacy behavior.
type AttemptCompletionBoundary string

const (
	AttemptCompletionProcessExit AttemptCompletionBoundary = "process_exit"
)

func (b AttemptCompletionBoundary) Valid() bool {
	return b == "" || b == AttemptCompletionProcessExit
}

// AdmissionSnapshot is the typed, immutable execution evidence retained on an
// AttemptSessionRef. Additional audit-only fields remain forward-compatible.
type AdmissionSnapshot struct {
	SnapshotVersion        int                            `json:"snapshotVersion"`
	Harness                string                         `json:"harness"`
	ModelSelection         ExecutionBindingModelSelection `json:"modelSelection"`
	RequestedModel         string                         `json:"requestedModel"`
	EffectiveModel         string                         `json:"effectiveModel"`
	WorkUnitID             string                         `json:"workUnitId"`
	Mode                   SessionMode                    `json:"mode"`
	RunBriefCoreDigest     string                         `json:"runBriefCoreDigest"`
	RunBriefCompiledDigest string                         `json:"runBriefCompiledDigest"`
	ExecutionPolicy        AttemptExecutionPolicy         `json:"executionPolicy"`
	ExecutionPolicyDigest  string                         `json:"executionPolicyDigest"`
	SessionID              string                         `json:"sessionId"`
	CompletionBoundary     AttemptCompletionBoundary      `json:"completionBoundary,omitempty"`
}

func ParseAdmissionSnapshot(raw string) (AdmissionSnapshot, error) {
	var snapshot AdmissionSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return AdmissionSnapshot{}, err
	}
	return snapshot, nil
}

func LooksLikeGovernedAdmissionSnapshot(raw string) bool {
	var envelope struct {
		SnapshotVersion       int             `json:"snapshotVersion"`
		ExecutionPolicy       json.RawMessage `json:"executionPolicy"`
		ExecutionPolicyDigest string          `json:"executionPolicyDigest"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return false
	}
	return envelope.SnapshotVersion == AdmissionSnapshotVersion &&
		len(envelope.ExecutionPolicy) > 0 && strings.TrimSpace(envelope.ExecutionPolicyDigest) != ""
}

// ValidateSession proves that the snapshot, session reference, and current
// session row describe the same governed execution.
func (s AdmissionSnapshot) ValidateSession(rec SessionRecord, ref AttemptSessionRef, marker string) error {
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("session evidence is invalid: %w", err)
	}
	if s.SnapshotVersion != AdmissionSnapshotVersion {
		return fmt.Errorf("admission snapshot version %d is unsupported", s.SnapshotVersion)
	}
	if s.SessionID != string(rec.ID) || ref.SessionID != string(rec.ID) {
		return fmt.Errorf("evidence is bound to a different session")
	}
	mode := NormalizeSessionMode(rec.Mode)
	if ref.Mode != mode || s.Mode != mode {
		return fmt.Errorf("evidence mode does not match session")
	}
	if ref.Harness != rec.Harness || s.Harness != string(rec.Harness) {
		return fmt.Errorf("evidence harness does not match session")
	}
	if s.WorkUnitID == "" || s.RunBriefCoreDigest == "" || s.RunBriefCompiledDigest == "" {
		return fmt.Errorf("admission snapshot is incomplete")
	}
	if ref.RunBriefCoreDigest != s.RunBriefCoreDigest || ref.RunBriefCompiledDigest != s.RunBriefCompiledDigest {
		return fmt.Errorf("run brief evidence does not match")
	}
	if err := s.ExecutionPolicy.Validate(); err != nil {
		return fmt.Errorf("execution policy is invalid: %w", err)
	}
	if s.ExecutionPolicy.RunBriefCoreDigest != s.RunBriefCoreDigest {
		return fmt.Errorf("execution policy is bound to a different RunBrief")
	}
	digest, err := s.ExecutionPolicy.Digest()
	if err != nil {
		return fmt.Errorf("execution policy digest: %w", err)
	}
	if digest != s.ExecutionPolicyDigest {
		return fmt.Errorf("execution policy digest does not match snapshot")
	}
	if marker != "" && marker != digest {
		return fmt.Errorf("execution policy digest does not match session marker")
	}
	if s.ModelSelection == ExecutionBindingModelExplicit && strings.TrimSpace(s.RequestedModel) == "" {
		return fmt.Errorf("explicit model is missing")
	}
	if s.ModelSelection == ExecutionBindingModelProviderDefault && strings.TrimSpace(s.RequestedModel) != "" {
		return fmt.Errorf("provider-default model is not empty")
	}
	if !s.CompletionBoundary.Valid() {
		return fmt.Errorf("completion boundary %q is unsupported", s.CompletionBoundary)
	}
	return nil
}
