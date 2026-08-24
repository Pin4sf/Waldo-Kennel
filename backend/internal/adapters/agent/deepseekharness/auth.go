package deepseekharness

import (
	"context"
	"os"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/authprobe"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

var _ ports.AgentAuthChecker = (*Plugin)(nil)

// AuthStatus returns the plugin's local authentication status. The env probe
// covers DeepSeek's documented API-key convention; everything else defers to
// the shared CLI probe against the resolved binary. The result is advisory
// only: a passing probe never guarantees a later spawn will succeed.
func (p *Plugin) AuthStatus(ctx context.Context) (ports.AgentAuthStatus, error) {
	binary, err := p.ResolveBinary(ctx)
	if err != nil {
		return ports.AgentAuthStatusUnknown, err
	}
	if status, ok := dshEnvAuthStatus(); ok {
		return status, nil
	}
	return authprobe.CLIStatus(ctx, binary, nil)
}

func dshEnvAuthStatus() (ports.AgentAuthStatus, bool) {
	for _, key := range []string{"DSH_API_KEY", "DEEPSEEK_API_KEY"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return ports.AgentAuthStatusAuthorized, true
		}
	}
	return "", false
}
