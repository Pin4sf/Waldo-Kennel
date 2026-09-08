package store_test

import (
	"time"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// admissionFor builds the admission packet these storage tests used to pass as
// loose arguments. An Attempt now names the WorkUnit it executes, so the plan's
// first unit is the one admitted.
func admissionFor(outcomeID domain.OutcomeID, plan domain.PlanRevision, requestKey, fenceSubject string) ports.AttemptAdmission {
	return admissionAt(outcomeID, plan, requestKey, fenceSubject, time.Now().UTC())
}

// admissionAt is admissionFor with an explicit clock, for tests that pin time.
func admissionAt(outcomeID domain.OutcomeID, plan domain.PlanRevision, requestKey, fenceSubject string, at time.Time) ports.AttemptAdmission {
	admission := ports.AttemptAdmission{
		OutcomeID:              outcomeID,
		PlanRevisionID:         plan.ID,
		ContractRevisionNumber: plan.ContractRevisionNumber,
		RequestKey:             requestKey,
		FenceSubject:           fenceSubject,
		At:                     at,
	}
	if len(plan.WorkUnits) > 0 {
		admission.WorkUnitID = plan.WorkUnits[0].ID
	}
	return admission
}
