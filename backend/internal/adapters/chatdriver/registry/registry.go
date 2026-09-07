// Package registry resolves the structured Chat driver for a provider harness.
//
// Registration is the capability gate: a harness with no driver here remains a
// worker/TUI provider and cannot be selected for coordinator-class chat flows.
package registry

import (
	"log/slog"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/claudecode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/codex"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/agent/opencode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/chatdriver/claudeacp"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/chatdriver/codexappserver"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/chatdriver/opencodeacp"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Registry maps a harness to its structured chat driver.
type Registry struct {
	drivers map[domain.AgentHarness]ports.ChatDriver
}

var _ ports.ChatDriverRegistry = (*Registry)(nil)

func New(drivers ...ports.ChatDriver) *Registry {
	byHarness := make(map[domain.AgentHarness]ports.ChatDriver, len(drivers))
	for _, driver := range drivers {
		if driver == nil {
			continue
		}
		byHarness[driver.Harness()] = driver
	}
	return &Registry{drivers: byHarness}
}

// Build returns the structured drivers Kennel ships. Codex uses app-server;
// Claude Code and OpenCode use ACP-backed transports. Cursor and Pi remain
// worker/TUI-only until they expose an equivalent structured protocol.
func Build(log *slog.Logger) *Registry {
	return New(
		codexappserver.New(codex.New(), log),
		claudeacp.New(claudecode.New(), log),
		opencodeacp.New(opencode.New(), log),
	)
}

func (r *Registry) Driver(harness domain.AgentHarness) (ports.ChatDriver, error) {
	driver, ok := r.drivers[harness]
	if !ok {
		return nil, ports.ErrChatUnsupported
	}
	return driver, nil
}

func (r *Registry) SupportsChat(harness domain.AgentHarness) bool {
	_, ok := r.drivers[harness]
	return ok
}

func (r *Registry) Harnesses() []domain.AgentHarness {
	out := make([]domain.AgentHarness, 0, len(r.drivers))
	for harness := range r.drivers {
		out = append(out, harness)
	}
	return out
}
