package sessionmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// attemptSessionEvidenceStore is deliberately optional so old ordinary
// sessions and small compatibility stores remain readable. Production SQLite
// implements it. A session marked as governed cannot recover without it.
type attemptSessionEvidenceStore interface {
	LatestAttemptSessionRefForSession(ctx context.Context, sessionID string) (domain.AttemptSessionRef, bool, error)
}

type admissionSnapshot struct {
	SnapshotVersion        int                                   `json:"snapshotVersion"`
	Harness                string                                `json:"harness"`
	ModelSelection         domain.ExecutionBindingModelSelection `json:"modelSelection"`
	RequestedModel         string                                `json:"requestedModel"`
	EffectiveModel         string                                `json:"effectiveModel"`
	WorkUnitID             string                                `json:"workUnitId"`
	Mode                   domain.SessionMode                    `json:"mode"`
	RunBriefCoreDigest     string                                `json:"runBriefCoreDigest"`
	RunBriefCompiledDigest string                                `json:"runBriefCompiledDigest"`
	ExecutionPolicy        domain.AttemptExecutionPolicy         `json:"executionPolicy"`
	ExecutionPolicyDigest  string                                `json:"executionPolicyDigest"`
	SessionID              string                                `json:"sessionId"`
}

type recoveryExecution struct {
	binding domain.ExecutionBinding
	policy  domain.AttemptExecutionPolicy
}

func (m *Manager) loadRecoveryExecution(ctx context.Context, rec domain.SessionRecord) (*recoveryExecution, error) {
	marker := strings.TrimSpace(rec.Metadata.GovernedExecutionPolicyDigest)
	store, supported := m.store.(attemptSessionEvidenceStore)
	if !supported {
		if marker == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("governed recovery evidence unavailable")
	}

	ref, found, err := store.LatestAttemptSessionRefForSession(ctx, string(rec.ID))
	if err != nil {
		return nil, fmt.Errorf("load governed recovery evidence: %w", err)
	}
	if !found {
		if marker == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("governed recovery evidence is missing")
	}
	if marker == "" && !looksLikeGovernedSnapshot(ref.AdmissionSnapshot) {
		// Older Attempt refs remain readable for inspection/recovery compatibility,
		// but they never acquire a synthesized policy on this path.
		return nil, nil
	}

	var snapshot admissionSnapshot
	if err := json.Unmarshal([]byte(ref.AdmissionSnapshot), &snapshot); err != nil {
		return nil, fmt.Errorf("governed recovery admission snapshot is invalid: %w", err)
	}
	if err := validateRecoveryEvidence(rec, ref, snapshot, marker); err != nil {
		return nil, err
	}
	binding := domain.ExecutionBinding{
		Provider:       domain.AgentHarness(snapshot.Harness),
		ModelSelection: snapshot.ModelSelection,
		Model:          strings.TrimSpace(snapshot.RequestedModel),
	}
	if err := binding.ValidateForNewWork(); err != nil {
		return nil, fmt.Errorf("governed recovery execution binding is invalid: %w", err)
	}
	return &recoveryExecution{binding: binding, policy: snapshot.ExecutionPolicy}, nil
}

func looksLikeGovernedSnapshot(raw string) bool {
	var envelope struct {
		SnapshotVersion       int             `json:"snapshotVersion"`
		ExecutionPolicy       json.RawMessage `json:"executionPolicy"`
		ExecutionPolicyDigest string          `json:"executionPolicyDigest"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return false
	}
	return envelope.SnapshotVersion == domain.AdmissionSnapshotVersion &&
		len(envelope.ExecutionPolicy) > 0 && strings.TrimSpace(envelope.ExecutionPolicyDigest) != ""
}

func validateRecoveryEvidence(
	rec domain.SessionRecord,
	ref domain.AttemptSessionRef,
	snapshot admissionSnapshot,
	marker string,
) error {
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("governed recovery session evidence is invalid: %w", err)
	}
	if snapshot.SnapshotVersion != domain.AdmissionSnapshotVersion {
		return fmt.Errorf("governed recovery admission snapshot version %d is unsupported", snapshot.SnapshotVersion)
	}
	if snapshot.SessionID != string(rec.ID) || ref.SessionID != string(rec.ID) {
		return fmt.Errorf("governed recovery evidence is bound to a different session")
	}
	mode := domain.NormalizeSessionMode(rec.Mode)
	if ref.Mode != mode || snapshot.Mode != mode {
		return fmt.Errorf("governed recovery evidence mode does not match session")
	}
	if ref.Harness != rec.Harness || snapshot.Harness != string(rec.Harness) {
		return fmt.Errorf("governed recovery evidence harness does not match session")
	}
	if snapshot.WorkUnitID == "" || snapshot.RunBriefCoreDigest == "" || snapshot.RunBriefCompiledDigest == "" {
		return fmt.Errorf("governed recovery admission snapshot is incomplete")
	}
	if ref.RunBriefCoreDigest != snapshot.RunBriefCoreDigest || ref.RunBriefCompiledDigest != snapshot.RunBriefCompiledDigest {
		return fmt.Errorf("governed recovery run brief evidence does not match")
	}
	if err := snapshot.ExecutionPolicy.Validate(); err != nil {
		return fmt.Errorf("governed recovery execution policy is invalid: %w", err)
	}
	if snapshot.ExecutionPolicy.RunBriefCoreDigest != snapshot.RunBriefCoreDigest {
		return fmt.Errorf("governed recovery policy is bound to a different RunBrief")
	}
	digest, err := snapshot.ExecutionPolicy.Digest()
	if err != nil {
		return fmt.Errorf("governed recovery execution policy digest: %w", err)
	}
	if digest != snapshot.ExecutionPolicyDigest {
		return fmt.Errorf("governed recovery execution policy digest does not match snapshot")
	}
	if marker != "" && marker != digest {
		return fmt.Errorf("governed recovery execution policy digest does not match session marker")
	}
	if snapshot.ModelSelection == domain.ExecutionBindingModelExplicit && strings.TrimSpace(snapshot.RequestedModel) == "" {
		return fmt.Errorf("governed recovery explicit model is missing")
	}
	if snapshot.ModelSelection == domain.ExecutionBindingModelProviderDefault && strings.TrimSpace(snapshot.RequestedModel) != "" {
		return fmt.Errorf("governed recovery provider-default model is not empty")
	}
	return nil
}

func recoveryAgentConfig(rec domain.SessionRecord, project domain.ProjectRecord, execution *recoveryExecution) (ports.AgentConfig, error) {
	if execution == nil {
		return effectiveAgentConfig(rec.Kind, project.Config), nil
	}
	binding := execution.binding
	policy := execution.policy
	cfg, _, err := prepareSpawnExecution(ports.SpawnConfig{
		Kind:                  rec.Kind,
		Harness:               binding.Provider,
		ExactExecutionBinding: &binding,
		ExecutionPolicy:       &policy,
	}, project.Config)
	if err != nil {
		return ports.AgentConfig{}, fmt.Errorf("resolve governed recovery config: %w", err)
	}
	return cfg.AgentConfig, nil
}
