// Package reviewer is the single source of truth for the code-review provider
// adapters Kennel ships. Reviewer identity is separate from worker identity, but
// both product surfaces intentionally use the same five first-class providers.
package reviewer

import (
	"fmt"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/reviewer/claudecode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/reviewer/codex"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/reviewer/cursor"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/reviewer/opencode"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/reviewer/pi"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// Adapter is a registered reviewer: a ports.Reviewer that names its harness.
type Adapter interface {
	ports.Reviewer
	Harness() domain.ReviewerHarness
}

// Constructors returns the complete first-class reviewer set shipped by Kennel.
func Constructors() []Adapter {
	return []Adapter{
		codex.New(),
		claudecode.New(),
		opencode.New(),
		cursor.New(),
		pi.New(),
	}
}

// Resolver maps a reviewer harness onto its adapter.
type Resolver struct {
	reviewers map[domain.ReviewerHarness]ports.Reviewer
}

var _ ports.ReviewerResolver = (*Resolver)(nil)

// NewResolver builds a Resolver from the shipped reviewer adapters. The domain
// vocabulary and registry must remain exactly aligned.
func NewResolver() (*Resolver, error) {
	m := make(map[domain.ReviewerHarness]ports.Reviewer)
	for _, adapter := range Constructors() {
		harness := adapter.Harness()
		if !harness.IsKnown() {
			return nil, fmt.Errorf("reviewer adapter %q is not in domain.AllReviewerHarnesses", harness)
		}
		if _, duplicate := m[harness]; duplicate {
			return nil, fmt.Errorf("reviewer harness %q is registered twice", harness)
		}
		m[harness] = adapter
	}
	for _, harness := range domain.AllReviewerHarnesses {
		if _, ok := m[harness]; !ok {
			return nil, fmt.Errorf("reviewer harness %q has no registered adapter", harness)
		}
	}
	return &Resolver{reviewers: m}, nil
}

func (r *Resolver) Reviewer(harness domain.ReviewerHarness) (ports.Reviewer, bool) {
	reviewer, ok := r.reviewers[harness]
	return reviewer, ok
}
