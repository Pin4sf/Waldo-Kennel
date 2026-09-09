package intelligence

import (
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

func TestDigestContractRequestIncludesRepositoryContext(t *testing.T) {
	base := ports.ContractIntelligenceRequest{
		Session: domain.IntakeSession{
			ID:        "intake-1",
			ProjectID: "project-1",
			Statement: "make the repository safer",
		},
		RepositoryContext: ports.RepositoryContextSnapshot{
			ProjectID: "project-1",
			Revision:  "rev-1",
			Files: []ports.RepositoryContextFile{{
				Path:    "README.md",
				Content: "first inspected fact",
			}},
		},
	}
	changed := base
	changed.RepositoryContext.Files = []ports.RepositoryContextFile{{
		Path:    "README.md",
		Content: "different inspected fact",
	}}

	first, err := digestContractRequest(base)
	if err != nil {
		t.Fatalf("digest base request: %v", err)
	}
	second, err := digestContractRequest(changed)
	if err != nil {
		t.Fatalf("digest changed request: %v", err)
	}
	if first == second {
		t.Fatalf("repository context did not affect contract input digest: %q", first)
	}
}
