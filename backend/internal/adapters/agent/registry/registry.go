// Package registry is the single source of truth for coding harness adapters
// shipped by Kennel. Runtime installation/authentication is discovered
// separately; this file defines product support only.
package registry

import (
	"fmt"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/claudecode"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/codex"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/cursor"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/opencode"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/pi"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// Constructors returns exactly the five integrations Kennel supports.
func Constructors() []adapters.Adapter {
	return []adapters.Adapter{
		codex.New(),
		claudecode.New(),
		opencode.New(),
		cursor.New(),
		pi.New(),
	}
}

func Build() (*adapters.Registry, error) {
	reg := adapters.NewRegistry()
	for _, adapter := range Constructors() {
		if err := reg.Register(adapter); err != nil {
			return nil, fmt.Errorf("register provider adapter %q: %w", adapter.Manifest().ID, err)
		}
	}
	return reg, nil
}

// HarnessAgent pairs a provider identity with the adapter that drives it.
type HarnessAgent struct {
	Harness  domain.AgentHarness
	Manifest adapters.Manifest
	Agent    ports.Agent
}

func Harnessed() []HarnessAgent {
	constructors := Constructors()
	out := make([]HarnessAgent, 0, len(constructors))
	for _, adapter := range constructors {
		agent, ok := adapter.(ports.Agent)
		if !ok {
			continue
		}
		out = append(out, HarnessAgent{
			Harness:  domain.AgentHarness(adapter.Manifest().ID),
			Manifest: adapter.Manifest(),
			Agent:    agent,
		})
	}
	return out
}
