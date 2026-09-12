package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/supervisorcap"
)

const supervisedExitReportTimeout = 5 * time.Second

func newAgentProcessCommand(ctx *commandContext) *cobra.Command {
	root := &cobra.Command{
		Use:    "agent-process",
		Short:  "Run an Kennel-managed agent process (internal)",
		Hidden: true,
	}
	root.AddCommand(newAgentProcessSuperviseCommand(ctx))
	return root
}

func newAgentProcessSuperviseCommand(ctx *commandContext) *cobra.Command {
	var sessionID string
	var launchID string
	cmd := &cobra.Command{
		Use:    "supervise --session <id> --launch <id> -- <command> [args...]",
		Short:  "Supervise one managed agent process (internal)",
		Hidden: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageError{fmt.Errorf("agent command is required")}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID = strings.TrimSpace(sessionID)
			launchID = strings.TrimSpace(launchID)
			if !sessionIDPattern.MatchString(sessionID) {
				return usageError{fmt.Errorf("invalid session id")}
			}
			if !sessionIDPattern.MatchString(launchID) {
				return usageError{fmt.Errorf("invalid launch id")}
			}
			ctx.runSupervisedProcess(cmd.Context(), sessionID, launchID, args)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "Kennel session id")
	cmd.Flags().StringVar(&launchID, "launch", "", "Kennel process launch id")
	return cmd
}

func (c *commandContext) runSupervisedProcess(ctx context.Context, sessionID, launchID string, argv []string) {
	child := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // argv is constructed by the selected agent adapter.
	child.Stdin = c.deps.In
	child.Stdout = c.deps.Out
	child.Stderr = c.deps.Err
	// The bearer authenticates this wrapper to the daemon. The provider child
	// must not inherit it or gain the ability to forge its own exit status.
	child.Env = environmentWithout(os.Environ(), supervisorcap.EnvCapability)

	if err := child.Start(); err != nil {
		_, _ = fmt.Fprintf(c.deps.Err, "ao: start managed agent: %v\n", err)
		c.reportSupervisedExit(sessionID, launchID, nil, "start_failed")
		return
	}

	// The child shares the terminal foreground process group and therefore
	// receives Ctrl-C directly. Consume the supervisor's copy so it remains
	// alive long enough to reap the child and publish the exit observation.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	waitErr := child.Wait()
	signal.Stop(interrupts)

	code, reason := supervisedExitFacts(ctx, child.ProcessState, waitErr)
	c.reportSupervisedExit(sessionID, launchID, code, reason)
}

func supervisedExitFacts(ctx context.Context, state *os.ProcessState, waitErr error) (*int, string) {
	if ctx.Err() != nil {
		return nil, "cancelled"
	}
	if state != nil {
		code := state.ExitCode()
		if code >= 0 {
			if code == 0 {
				return &code, "exited"
			}
			return &code, "failed"
		}
	}
	if waitErr != nil {
		return nil, "failed"
	}
	return nil, "unknown"
}

func environmentWithout(env []string, name string) []string {
	prefix := name + "="
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (c *commandContext) reportSupervisedExit(sessionID, launchID string, exitCode *int, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), supervisedExitReportTimeout)
	defer cancel()
	path := "/api/v1/sessions/" + sessionID + "/activity"
	req := setActivityAPIRequest{State: "exited", Event: "process-exited", LaunchID: launchID}
	headers := map[string]string(nil)
	if token := strings.TrimSpace(os.Getenv(supervisorcap.EnvCapability)); token != "" {
		req.ProcessExit = &supervisedProcessExitRequest{ExitCode: exitCode, Reason: reason}
		headers = map[string]string{supervisorcap.HeaderCapability: token}
	}
	if err := c.doJSONPathWithHeaders(ctx, http.MethodPost, path, req, nil, headers); err != nil {
		// Reconciliation can still report process absence. Keep the delivery
		// failure visible without preventing the terminal's shell.
		c.reportHookFailure("agent-process", "process-exited", sessionID, err)
	}
}
