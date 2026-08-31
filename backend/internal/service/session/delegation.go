package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	sessionmanager "github.com/aoagents/agent-orchestrator/backend/internal/session_manager"
)

const (
	delegatedTaskTitleLimit             = 20
	delegatedTaskUntitledName           = "Untitled task"
	delegatedTaskTitleRefinementTimeout = time.Minute
	outcomeSessionTitleLimit            = 80
)

// DelegateTaskInput describes a task AO should spawn as a worker session. Brief
// may be empty to open an idle worker that the user can instruct later. Empty
// RequestedAgent means the spawn uses the project's worker-agent default.
type DelegateTaskInput struct {
	ProjectID      domain.ProjectID
	Brief          string
	Outcome        bool
	RequestedAgent domain.AgentHarness
	Model          string
	RequestedMode  domain.SessionMode
	Attachments    []ports.SpawnAttachment
}

// DelegateTaskOutcome identifies the spawned worker. OrchestratorID remains
// optional for wire compatibility; asynchronous title refinement does not wait
// to resolve the coordinator before returning.
type DelegateTaskOutcome struct {
	OrchestratorID domain.SessionID
	WorkerID       domain.SessionID
}

// DelegateTask spawns the worker directly, matching `kennel spawn`, with a
// provisional display name derived from the task brief. AO then best-effort
// refines that title in the background through the project orchestrator,
// resuming or creating the coordinator when necessary.
func (s *Service) DelegateTask(ctx context.Context, in DelegateTaskInput) (DelegateTaskOutcome, error) {
	if _, err := s.requireProject(ctx, in.ProjectID); err != nil {
		return DelegateTaskOutcome{}, err
	}
	if in.RequestedAgent != "" && !in.RequestedAgent.IsSelectableForNewWork() {
		return DelegateTaskOutcome{}, apierr.Invalid("HARNESS_NOT_SELECTABLE", "Requested agent is not selectable for new work", nil)
	}
	if in.RequestedMode != "" && !in.RequestedMode.Valid() {
		return DelegateTaskOutcome{}, apierr.Invalid("INVALID_SESSION_MODE", "mode must be chat or tui", nil)
	}
	if in.Outcome {
		return s.submitOutcome(ctx, in)
	}
	prompt := in.Brief
	if strings.TrimSpace(prompt) == "" {
		prompt = ""
	}

	worker, _, _, err := s.manager.Spawn(ctx, ports.SpawnConfig{
		ProjectID:     in.ProjectID,
		Kind:          domain.KindWorker,
		Harness:       in.RequestedAgent,
		Prompt:        prompt,
		DisplayName:   delegatedTaskDisplayName(in.Brief),
		AgentConfig:   ports.AgentConfig{Model: strings.TrimSpace(in.Model)},
		RequestedMode: in.RequestedMode,
		Attachments:   in.Attachments,
	})
	if err != nil {
		return DelegateTaskOutcome{}, toAPIError(err)
	}

	// The worker spawn is the commit point. Coordinator startup and title
	// generation must never hold the new-task response open. A promptless worker
	// stays idle with its provisional title until the user supplies instructions.
	if prompt != "" {
		s.refineDelegatedTaskTitleInBackground(worker.ID, in)
	}
	return DelegateTaskOutcome{WorkerID: worker.ID}, nil
}

func (s *Service) submitOutcome(ctx context.Context, in DelegateTaskInput) (DelegateTaskOutcome, error) {
	brief := strings.TrimSpace(in.Brief)
	if brief == "" {
		return DelegateTaskOutcome{}, apierr.Invalid("OUTCOME_REQUIRED", "Describe the outcome you want Kennel to deliver", nil)
	}
	orchestratorID, err := s.taskTitleOrchestrator(ctx, in.ProjectID)
	if err != nil {
		return DelegateTaskOutcome{}, toAPIError(err)
	}
	if err := s.manager.WaitForMessageDeliveryReady(ctx, orchestratorID); err != nil {
		return DelegateTaskOutcome{}, toAPIError(fmt.Errorf("wait for outcome orchestrator %s: %w", orchestratorID, err))
	}
	attachmentPaths, err := s.manager.StageAttachments(ctx, orchestratorID, in.Attachments)
	if err != nil {
		return DelegateTaskOutcome{}, toAPIError(fmt.Errorf("stage outcome attachments: %w", err))
	}
	message := outcomeIntakeMessage(in)
	if len(attachmentPaths) > 0 {
		message += "\n\nFiles attached to this outcome:\n- " + strings.Join(attachmentPaths, "\n- ")
	}
	orchestrator, ok, err := s.store.GetSession(ctx, orchestratorID)
	if err != nil {
		return DelegateTaskOutcome{}, toAPIError(fmt.Errorf("load outcome orchestrator %s: %w", orchestratorID, err))
	}
	if !ok {
		return DelegateTaskOutcome{}, apierr.NotFound("SESSION_NOT_FOUND", "Outcome orchestrator was not found")
	}
	outcomeTitle := outcomeSessionDisplayName(brief)
	renamed, err := s.store.RenameSession(ctx, orchestratorID, outcomeTitle, s.now().UTC())
	if err != nil {
		return DelegateTaskOutcome{}, toAPIError(fmt.Errorf("record outcome on orchestrator %s: %w", orchestratorID, err))
	}
	if !renamed {
		return DelegateTaskOutcome{}, apierr.NotFound("SESSION_NOT_FOUND", "Outcome orchestrator was not found")
	}
	if err := s.manager.Send(ctx, orchestratorID, message, nil); err != nil {
		// The Outcome was not delivered, so do not leave durable UI claiming that
		// it was. Restoration is best-effort because the send error remains primary.
		_, _ = s.store.RenameSession(context.WithoutCancel(ctx), orchestratorID, orchestrator.DisplayName, s.now().UTC())
		return DelegateTaskOutcome{}, toAPIError(fmt.Errorf("send outcome to %s: %w", orchestratorID, err))
	}
	// WorkerID remains populated for wire compatibility with the existing
	// delegate response. In outcome mode it identifies the coordinating session;
	// no implementation worker exists until the user approves the deliverables.
	return DelegateTaskOutcome{WorkerID: orchestratorID, OrchestratorID: orchestratorID}, nil
}

func outcomeIntakeMessage(in DelegateTaskInput) string {
	var b strings.Builder
	b.WriteString("KENNEL OUTCOME INTAKE\n\n")
	b.WriteString("The user wants this outcome:\n")
	b.WriteString(strings.TrimSpace(in.Brief))
	b.WriteString("\n\nDo not spawn workers or begin implementation yet. First decide whether the outcome is sufficiently precise. Ask only clarifying questions that remove material ambiguity.\n\n")
	b.WriteString("If questions are needed, end with KENNEL_OUTCOME_QUESTIONS_JSON: followed on the same line by valid JSON shaped as {\"questions\":[{\"id\":\"stable-id\",\"prompt\":\"question\",\"options\":[{\"id\":\"option-id\",\"label\":\"short answer\",\"description\":\"tradeoff\",\"recommended\":true}]}]}. Provide 2-4 relevant options per question and no Other option.\n\n")
	b.WriteString("Once clear, end with KENNEL_OUTCOME_PLAN_JSON: followed on the same line by valid JSON shaped as {\"summary\":\"concise plan\",\"deliverables\":[{\"id\":\"stable-id\",\"title\":\"deliverable\",\"description\":\"scope\",\"agent\":\"installed harness id\",\"checks\":[\"objective check\"]}],\"constraints\":[\"assumption or exclusion\"]}. Every deliverable needs an agent and objective checks.\n\n")
	b.WriteString("Only KENNEL_OUTCOME_PLAN_APPROVED: is approval. KENNEL_OUTCOME_PLAN_REVISION: requests a replacement plan. You must not delegate or implement before approval. After approval, create the worker sessions, coordinate them, verify every check with evidence, and report completion on the Kanban board. End every Outcome progress update with exactly one current lifecycle marker: KENNEL_OUTCOME_STATUS: working while implementing, KENNEL_OUTCOME_STATUS: needs_you when a user decision is required, KENNEL_OUTCOME_STATUS: reviewing while validating deliverables and checks, or KENNEL_OUTCOME_STATUS: ready_to_merge only when every approved check has evidence.\n")
	if in.RequestedAgent != "" {
		b.WriteString("\nPreferred worker harness after approval: ")
		b.WriteString(string(in.RequestedAgent))
	}
	if model := strings.TrimSpace(in.Model); model != "" {
		b.WriteString("\nPreferred worker model after approval: ")
		b.WriteString(model)
	}
	return b.String()
}

func (s *Service) refineDelegatedTaskTitleInBackground(workerID domain.SessionID, in DelegateTaskInput) {
	work := func() {
		base := s.backgroundContext
		if base == nil {
			base = context.Background()
		}
		ctx, cancel := context.WithTimeout(base, delegatedTaskTitleRefinementTimeout)
		defer cancel()

		if err := s.refineDelegatedTaskTitle(ctx, workerID, in); err != nil && s.logger != nil {
			s.logger.Warn("delegated task title refinement failed",
				"projectID", in.ProjectID,
				"workerID", workerID,
				"error", err,
			)
		}
	}
	if s.runBackground != nil {
		s.runBackground(work)
		return
	}
	go work()
}

func (s *Service) refineDelegatedTaskTitle(ctx context.Context, workerID domain.SessionID, in DelegateTaskInput) error {
	orchestratorID, err := s.taskTitleOrchestrator(ctx, in.ProjectID)
	if err != nil {
		return err
	}
	if err := s.manager.WaitForMessageDeliveryReady(ctx, orchestratorID); err != nil {
		return fmt.Errorf("wait for title orchestrator %s: %w", orchestratorID, err)
	}
	if err := s.manager.Send(ctx, orchestratorID, taskTitleDelegationMessage(workerID, in), nil); err != nil {
		return fmt.Errorf("send title request to %s: %w", orchestratorID, err)
	}
	return nil
}

func (s *Service) taskTitleOrchestrator(ctx context.Context, projectID domain.ProjectID) (domain.SessionID, error) {
	unlock := s.lockOrchestratorProject(projectID)
	orchestrators, err := s.activeOrchestrators(ctx, projectID)
	if err != nil {
		unlock()
		return "", fmt.Errorf("list project orchestrators: %w", err)
	}

	running := make([]domain.Session, 0, len(orchestrators))
	for _, orchestrator := range orchestrators {
		if orchestrator.Activity.State != domain.ActivityExited {
			running = append(running, orchestrator)
		}
	}
	if len(running) > 0 {
		orchestratorID := newestSession(running).ID
		unlock()
		return orchestratorID, nil
	}
	if len(orchestrators) > 0 {
		orchestratorID := newestSession(orchestrators).ID
		_, resumeErr := s.manager.ResumeAgentWithMode(ctx, orchestratorID)
		unlock()
		if resumeErr != nil && !errors.Is(resumeErr, sessionmanager.ErrAgentNotExited) {
			return "", fmt.Errorf("resume project orchestrator %s: %w", orchestratorID, resumeErr)
		}
		return orchestratorID, nil
	}
	unlock()

	orchestrator, err := s.SpawnOrchestrator(ctx, projectID, false, "")
	if err != nil {
		return "", fmt.Errorf("start project orchestrator: %w", err)
	}
	return orchestrator.ID, nil
}

func delegatedTaskDisplayName(brief string) string {
	title := strings.Join(strings.Fields(brief), " ")
	if title == "" {
		return delegatedTaskUntitledName
	}
	if utf8.RuneCountInString(title) <= delegatedTaskTitleLimit {
		return title
	}
	return strings.TrimSpace(string([]rune(title)[:delegatedTaskTitleLimit]))
}

func outcomeSessionDisplayName(brief string) string {
	title := "Outcome: " + strings.Join(strings.Fields(brief), " ")
	if utf8.RuneCountInString(title) <= outcomeSessionTitleLimit {
		return title
	}
	return strings.TrimSpace(string([]rune(title)[:outcomeSessionTitleLimit]))
}

func taskTitleDelegationMessage(workerID domain.SessionID, in DelegateTaskInput) string {
	var b strings.Builder
	b.WriteString("Kennel TASK TITLE UPDATE\n")
	b.WriteString("A worker was already spawned directly with the user's task. Do not spawn another worker or orchestrator, and do not implement the task in this orchestrator session.\n")
	b.WriteString("Choose a concise task title from the brief and run:\n\n")
	b.WriteString("kennel session rename ")
	b.WriteString(string(workerID))
	b.WriteString(" \"<title, max 20 chars>\"\n\n")
	b.WriteString("Worker session id: ")
	b.WriteString(string(workerID))
	b.WriteString("\nTask brief:\n")
	b.WriteString(in.Brief)
	if model := strings.TrimSpace(in.Model); model != "" {
		b.WriteString("\nRequested model: ")
		b.WriteString(model)
	}
	return b.String()
}
