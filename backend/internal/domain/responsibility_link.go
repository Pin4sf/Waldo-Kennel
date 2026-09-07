package domain

import (
	"fmt"
	"strings"
	"time"
)

// ResponsibilityLinkID identifies explicit Home-to-Work lineage.
type ResponsibilityLinkID string

// IsZero reports whether the link identity is blank.
func (id ResponsibilityLinkID) IsZero() bool   { return strings.TrimSpace(string(id)) == "" }
func (id ResponsibilityLinkID) String() string { return string(id) }

// ResponsibilityLinkCreator identifies who explicitly created or ended lineage.
type ResponsibilityLinkCreator string

// ResponsibilityLinkCreatorOwner is the only creator currently authorized.
const ResponsibilityLinkCreatorOwner ResponsibilityLinkCreator = "owner"

// ResponsibilityLink is explicit lineage between a confirmed Home Open Loop
// and a Work Outcome. It carries no lifecycle authority over either side.
type ResponsibilityLink struct {
	ID                   ResponsibilityLinkID
	SourceOpenLoopID     OpenLoopID
	DestinationOutcomeID OutcomeID
	Creator              ResponsibilityLinkCreator
	Reason               string
	CreatedAt            time.Time
	EndedAt              *time.Time
	EndedBy              ResponsibilityLinkCreator
	EndedReason          string
}

// Validate checks explicit lineage without coupling responsibility lifecycles.
func (link ResponsibilityLink) Validate() error {
	if link.ID.IsZero() || link.SourceOpenLoopID.IsZero() || link.DestinationOutcomeID.IsZero() {
		return fmt.Errorf("responsibility link identity and endpoints are required")
	}
	if link.Creator != ResponsibilityLinkCreatorOwner {
		return fmt.Errorf("responsibility link requires an explicit owner creator")
	}
	if strings.TrimSpace(link.Reason) == "" || link.CreatedAt.IsZero() {
		return fmt.Errorf("responsibility link reason and created time are required")
	}
	if link.EndedAt == nil {
		if link.EndedBy != "" || strings.TrimSpace(link.EndedReason) != "" {
			return fmt.Errorf("active responsibility link cannot carry end metadata")
		}
		return nil
	}
	if link.EndedBy != ResponsibilityLinkCreatorOwner || strings.TrimSpace(link.EndedReason) == "" {
		return fmt.Errorf("ended responsibility link requires owner and reason")
	}
	if link.EndedAt.Before(link.CreatedAt) {
		return fmt.Errorf("responsibility link cannot end before creation")
	}
	return nil
}

// End returns a terminal lineage value without mutating either responsibility
// or the original link value.
func (link ResponsibilityLink) End(actor ResponsibilityLinkCreator, reason string, at time.Time) (ResponsibilityLink, error) {
	if link.EndedAt != nil {
		return ResponsibilityLink{}, fmt.Errorf("responsibility link is already ended")
	}
	ended := link
	ended.EndedAt = &at
	ended.EndedBy = actor
	ended.EndedReason = strings.TrimSpace(reason)
	if err := ended.Validate(); err != nil {
		return ResponsibilityLink{}, err
	}
	return ended, nil
}
