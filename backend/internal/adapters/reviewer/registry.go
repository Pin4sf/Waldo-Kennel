// Package reviewer is the single source of truth for code-review adapters
// shipped by Kennel.
package reviewer

import (
	"fmt"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/reviewer/claudecode"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/reviewer/codex"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/reviewer/cursor"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/reviewer/opencode"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/reviewer/pi"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

type Adapter interface {
	ports.Reviewer
	Harness() domain.ReviewerHarness
}

func Constructors() []Adapter {
	return []Adapter{
		codex.New(),
		claudecode.New(),
		opencode.New(),
		cursor.New(),
		pi.New(),
	}
}

type Resolver struct {
	reviewers map[domain.ReviewerHarness]ports.Reviewer
}

var _ ports.ReviewerResolver = (*Resolver)(nil)

func NewResolver() (*Resolver, error) {
	registered := make(map[domain.ReviewerHarness]ports.Reviewer)
	for _, adapter := range Constructors() {
		harness := adapter.Harness()
		if !harness.IsKnown() {
			return nil, fmt.Errorf("reviewer adapter %q is not an active Kennel provider", harness)
		}
		if _, duplicate := registered[harness]; duplicate {
			return nil, fmt.Errorf("reviewer harness %q is registered twice", harness)
		}
		registered[harness] = adapter
	}
	for _, harness := range domain.AllReviewerHarnesses {
		if _, ok := registered[harness]; !ok {
			return nil, fmt.Errorf("reviewer harness %q has no registered adapter", harness)
		}
	}
	return &Resolver{reviewers: registered}, nil
}

func (r *Resolver) Reviewer(harness domain.ReviewerHarness) (ports.Reviewer, bool) {
	reviewer, ok := r.reviewers[harness]
	return reviewer, ok
}
