package deepseekharness

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	aoprocess "github.com/aoagents/agent-orchestrator/backend/internal/process"
)

var _ ports.AgentProfileReadinessChecker = (*Plugin)(nil)

// profileProbeTimeout bounds the profile-composition probe. `dsh --profile
// <name> --dump-config` prints the composed profile tree and exits (verified
// launcher contract), so a healthy probe is fast; anything slower means the
// CLI or the profile is stuck and the honest answer is "not ready". A var
// rather than a const so tests can shrink the budget instead of waiting on a
// genuinely stuck child.
var profileProbeTimeout = 10 * time.Second

// ProfileReadiness reports whether the configured dsh profile can actually be
// composed. This replaces any credential-based claim: an API key in the
// environment cannot establish that an arbitrary profile — possibly pointed at
// a different provider — is launchable, so authentication plays no part here.
// Spawn remains the authoritative validation point.
func (p *Plugin) ProfileReadiness(ctx context.Context, cfg ports.AgentConfig) (ports.AgentProfileReadiness, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentProfileReadiness{}, err
	}
	profile := strings.TrimSpace(cfg.Profile)
	if profile == "" {
		return ports.AgentProfileReadiness{
			Ready:  false,
			Detail: "no dsh profile selected — create one ('dsh plugin --profile <name> add <package>') and set it as this agent's profile",
		}, nil
	}

	binary, err := p.ResolveBinary(ctx)
	if err != nil {
		return ports.AgentProfileReadiness{}, err
	}

	probeCtx, cancel := context.WithTimeout(ctx, profileProbeTimeout)
	defer cancel()
	output, err := aoprocess.CommandContext(probeCtx, binary, "--profile", profile, "--dump-config").CombinedOutput()
	if err != nil {
		// A probe killed by its own budget is a distinct truth from a CLI that
		// failed: report the timeout instead of dressing the kill up as an
		// exit-status failure.
		if probeCtx.Err() != nil {
			return ports.AgentProfileReadiness{
				Ready:  false,
				Detail: fmt.Sprintf("dsh --profile %s --dump-config timed out after %s", profile, profileProbeTimeout),
			}, nil
		}
		detail := strings.TrimSpace(string(output))
		if exitErr, ok := err.(*exec.ExitError); ok && detail == "" {
			detail = "profile composition failed with exit status " + exitErr.Error()
		}
		if len(detail) > 400 {
			detail = detail[len(detail)-400:]
		}
		return ports.AgentProfileReadiness{
			Ready:  false,
			Detail: "dsh --profile " + profile + " --dump-config failed: " + detail,
		}, nil
	}
	return ports.AgentProfileReadiness{
		Ready:  true,
		Detail: "dsh profile " + profile + " composed successfully",
	}, nil
}
