# Rework loop contract — 11 September 2026

Stable frontend contract for the owner loop that runs from rejecting a result
to accepting a revised one: `POST /api/v1/outcomes/{outcomeId}/acceptance-decisions`,
`GET /api/v1/outcomes/{outcomeId}/proof` and `GET /api/v1/outcomes/{outcomeId}/run`.

Generated sources are authoritative: `backend/internal/httpd/apispec/openapi.yaml`
and `frontend/src/api/schema.ts`. This note explains the behaviour.

## What changed

One additive response field and two additive reason codes. No field was
removed, renamed or retyped, and no request shape changed.

### `proof.activeCorrectionId` (new, string, omitted when absent)

`proof.corrections` is append-only history — every correction the owner has
ever recorded against the current Contract revision. `activeCorrectionId` names
the one still standing: the correction attached to the decision that set the
current `proofHorizon`. A renderer showing the whole list would keep asking for
changes that have already been made.

Absent means no rework or reopen stands against the current Contract revision.

### New action refusal reasons on `runState.eligibleActions`

| Code | Applies to | Meaning |
|---|---|---|
| `plan_revision_required` | `start` | The standing correction names the current Plan. Running that same approved Plan again would reproduce the result the owner rejected. `propose_plan` becomes available instead. |
| `contract_revision_required` | `start` | The standing correction names the Contract. Nothing below it can be revised while the agreement itself is what is wrong. |

### `runState.blocker.detail` on `rework_required`

When `attentionReason` is `rework_required`, the blocker's existing free-form
`detail` map now carries the standing correction, so the Mission can say what
has to change and in the owner's own words rather than only "rework required":

```json
{ "decisionId": "acc-…", "targetType": "plan", "targetId": "plan-…",
  "feedback": "the Plan itself is wrong" }
```

`targetId` is omitted when the correction records none. `detail` is absent when
no correction stands.

## Requesting rework ends the run authorization

This is the behaviour the checkpoint was missing, and the reason the loop could
not complete.

A `request_rework` or `reopen` decision now appends a **cancel** generation to
the Outcome's durable run intent, if one was in force. Rejecting a result is the
owner saying it is not the result they want; leaving the previous authorization
standing had two consequences, both wrong:

- the daemon would keep admitting WorkUnits against a result just rejected;
- because Start does not apply to an already-running intent, the owner could not
  authorize the revised work at all — `start` came back
  `run_already_authorized` and nothing could move.

Cancel is the existing "stop, and require a fresh Start" transition, not a new
state. Kennel never re-authorizes on the owner's behalf; it only stops claiming
they already did. After rework the renderer should expect:

- `intent.desired` = `cancelled` (or no intent, if none was in force);
- `state` = `needs_you`, `attentionReason` = `rework_required`;
- `start` available for an `attempt` or `work_unit` correction;
- `start` refused, `propose_plan` available, for a `plan` correction;
- `start` refused for a `contract` correction.

`accept` does not halt anything: accepting is not rejecting, and an Outcome with
nothing left to run has no authorization worth cancelling.

### Idempotency

The halt's replay identity is the acceptance decision, so retrying a correction
whose decision committed but whose halt failed **finishes** it rather than
appending a second cancellation. Sending the same `requestKey` twice yields one
decision and one intent generation. A concurrent owner command that moved the
intent first is left alone — it cannot have moved it into a state that admits
work, since only Start does that.

## Corrections resolve themselves

There is no "correction addressed" record to write, and a renderer should not
look for one:

- a `plan` correction stops naming the current Plan as soon as a fresh Plan is
  proposed, so `start` reopens on its own;
- a `contract` correction is bound to the Contract revision it was made against,
  and `proof.corrections` only ever reports corrections for the current
  revision — so revising the Contract removes it entirely.

## Fresh proof, preserved history

`proofHorizon` moves to the rework decision's timestamp. Evidence and
Verification recorded at or before it no longer make a criterion ready, so the
WorkUnit becomes retryable and the revised run must produce its own proof
against its own Attempt and artifact version.

Nothing is deleted. The rejected Attempt, its receipt and artifact version, the
evidence it produced, the decision that rejected it and the recorded feedback
all remain readable. Acceptance stays the owner's alone: fresh proof returns the
Outcome to `ready_for_review`, and never to `accepted`.
