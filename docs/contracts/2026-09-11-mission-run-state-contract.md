# Mission run-state contract — 11 September 2026

Stable frontend contract for `GET /api/v1/outcomes/{outcomeId}/run`,
`GET /api/v1/projects/{id}/outcome-run-states` and
`POST /api/v1/outcomes/{outcomeId}/run`. It repairs KUX-002: the projection and
the run-command endpoint now answer to one transition policy, so the Mission
can no longer offer a move the daemon rejects.

Generated sources are authoritative: `backend/internal/httpd/apispec/openapi.yaml`
and `frontend/src/api/schema.ts`. This note explains what the fields mean.

## What changed

One additive response field and three additive reason codes. No field was
removed, renamed or retyped, and no request shape changed. A renderer built
against the previous contract keeps working; it will simply keep showing the
old, wrong Start affordance until it reads the codes below.

### `runState.intent.bindsCurrentPlan` (new, boolean, always present)

False when the authorization names a Plan revision the Outcome has moved past.
Continuation schedules against the Plan the *intent* names, so a superseded
authorization admits nothing however runnable the current Plan looks. The only
way forward is Cancel, then Start against the current Plan.

### New `attentionReason` / `blocker.code` values

| Code | Meaning | Owner's next move |
|---|---|---|
| `run_paused` | Durable run intent is `paused` and nothing is in flight. | Resume, or Cancel. Never Start-as-primary: Resume is the command. |
| `run_intent_plan_superseded` | The authorized run names a Plan revision the Outcome has moved past. `blocker.detail.authorizedPlanId` names it. | Cancel, then authorize the current Plan. |

### New action refusal reason

| Code | Applies to | Meaning |
|---|---|---|
| `run_already_authorized` | `start` | Durable run intent already authorizes continuation, so Start has nothing left to authorize. `POST /run` with `action=start` returns 409 `RUN_ACTION_UNAVAILABLE` here. |

## The invariant a renderer may now rely on

For the four run-intent commands — `start`, `pause`, `resume`, `cancel` — an
action reported `available: true` is one `POST /outcomes/{id}/run` will accept
on the same durable state, and an action reported `available: false` is one it
will refuse. Both sides ask `domain.NextRunIntent`; the projection additionally
applies the Plan/gate/Attempt preconditions `CommandRun` applies.

The remaining refusals are genuinely client-side and stay the caller's
responsibility, because the projection cannot know them in advance:

- `RUN_INTENT_STALE` — the caller sent an `expectedGeneration` or
  `expectedContractRevision` that has since moved. Send the `generation` from
  the `intent` in the same response you rendered.
- `RUN_CUSTODY_UNKNOWN` — an Attempt's runtime status is unaccountable. Only a
  queued or running Attempt can be, and that already reports `attempt_active`.

## State projection between WorkUnits

This is the behaviour KUX-002 got wrong. With no Attempt in flight:

| Durable intent | Something runnable | `state` | `attentionReason` |
|---|---|---|---|
| none recorded / `idle` | yes | `needs_you` | `start_required` |
| `running`, binds current Plan | yes | `in_progress` | — |
| `running`, binds current Plan | no | `needs_you` | the schedule's own reason |
| `running`, superseded Plan | either | `needs_you` | `run_intent_plan_superseded` |
| `paused` | either | `needs_you` | `run_paused` |
| `cancelled` | yes | `needs_you` | `start_required` |

`in_progress` between WorkUnits is not a claim that a provider is executing. It
is the truthful statement that the owner has authorized continuation and the
daemon admits the next unit itself, so there is nothing for the owner to do.
`activeAttemptId` remains the field that says whether an Attempt exists.

## Pause, Resume and Cancel

- `pause` is offered exactly when intent is `running`.
- `resume` is offered exactly when intent is `paused` **and** the Plan is
  approved and still binds the Contract, because `resume` re-authorizes
  execution and re-validates the Plan exactly as `start` does. Against a stale
  Plan it is refused with `plan_no_longer_binds_current_contract`.
- `cancel` is offered when intent is `running` or `paused`, and also whenever a
  live Attempt exists — `CancelAttempt` is its own path, so a daemon with no
  run-intent storage can still stop one Attempt.

A daemon without run-intent storage reports `run_intent_unavailable` for
`pause` and `resume`; a wired daemon that simply has no intent yet reports
`no_active_run`. The two are different facts and only the first is a wiring
fault.
