// Package registry is the single source of truth for the first-class coding
// providers Kennel ships. Machine availability is discovered separately; this
// registry answers only which integrations are part of the product build.
package registry

import (
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/claudecode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/codex"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/cursor"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/opencode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/pi"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Constructors returns a fresh instance of every first-class provider adapter
// in stable display order. Adding another provider is a deliberate product
// decision: its domain identity, readiness probes and role capabilities must
// land with this registry change rather than being inherited as dormant code.
func Constructors() []adapters.Adapter {
	return []adapters.Adapter{
		codex.New(),
		claudecode.New(),
		opencode.New(),
		cursor.New(),
		pi.New(),
	}
}

// Build returns a registry populated with the shipped provider adapters.
func Build() (*adapters.Registry, error) {
	reg := adapters.NewRegistry()
	for _, adapter := range Constructors() {
		if err := reg.Register(adapter); err != nil {
			return nil, fmt.Errorf("register agent adapter %q: %w", adapter.Manifest().ID, err)
		}
	}
	return reg, nil
}

// HarnessAgent pairs a session harness with the adapter that drives it.
type HarnessAgent struct {
	Harness  domain.AgentHarness
	Manifest adapters.Manifest
	Agent    ports.Agent
}

// Harnessed returns every shipped adapter that implements ports.Agent, in the
// same stable order as Constructors.
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
