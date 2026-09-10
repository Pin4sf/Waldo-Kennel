# Kennel Work guide

The user manages Outcomes. Kennel manages their execution. Provider sessions are
technical detail beneath an Outcome, not the responsibility itself.

## Open and review an Outcome

Open Work, choose a Project and describe the result and observable success criteria.
Use Outcomes to browse the portfolio as Board or List. Search and Project filtering
narrow the list; contributing Outcomes are hidden until explicitly included.

Selecting a direct Outcome opens Mission beside the portfolio on wide windows.
Adjust Mission width with the slider (arrow keys work), or choose Expand Mission.
Narrow windows show Mission with Project/Outcome navigation still available on the
left. Back to Outcomes returns to the mounted portfolio and its filters.

Mission keeps the selected title, Project, Contract/Plan revision and connection
status visible. Contract shows the desired result, criteria, constraints, permission
ceiling, review requirements and pause conditions. Plan, Execution, Evidence & result,
and Decision history are views inside the same Mission.

## Configure reasoning and authorize

Waldo reasoning uses the owner's configured provider/model and credential. In Plan,
unavailable reasoning is explained before proposal. Expand Waldo reasoning to use the
existing settings controls without leaving Mission. A saved credential and verified
readiness are different facts; Kennel does not silently choose another provider.

Review all WorkUnits and dependencies in Graph or Table. Select a unit to inspect its
expected output, checks, capabilities, stops, proof readiness and Attempt history.
Graph arrows support keyboard selection. Branching dependencies do not imply parallel
execution; the daemon enforces serial custody.

Approve authorizes the exact Plan revision. Start is separate, in Execution. A Plan
bound to an older Contract cannot be approved from the current screen. Request revised
Plan records your feedback through the existing proposal API. If a response is lost,
feedback remains and resubmission is withheld: refresh and review the current Plan
before deciding whether another request is needed.

## Observe and review

Execution shows daemon schedule and Attempt facts. Start and available cancellation
or recovery actions use existing daemon admission. Durable Outcome pause/continue is
not exposed yet. A quiet provider does not prove that execution stopped.

Recorded usage lists reported tokens for sessions bound to this Outcome, preserving
unknown and incomplete reports. It does not claim total planning usage or cost.
Terminal inspection is optional, available only after an Attempt has a session.

Evidence & result reuses the current criterion evidence, verification and explicit
owner acceptance/rework controls. Provider completion and green checks do not accept
an Outcome. Decision history shows recorded owner decisions and Contract revisions.
Disconnected updates are visibly marked; new admission is withheld until the
stream reconnects. Existing Attempt cancel and containment remain available through
HTTP under daemon validation, even if the Plan is stale. Reconnection refetches current Mission facts.

## Current limits

The portfolio currently displays Contract/Plan authority facts. It does not yet offer
complete execution/acceptance status columns. Supplied-document selection, retained
artifact browsing and durable export require backend interfaces still under development.
Acceptance does not export, merge, publish or deploy files. No export success is implied.

Focused Work is the default. Home and standalone Waldo chat are hidden; existing data
is retained. `VITE_KENNEL_WORK_LAUNCH=0` opts out for development. Island startup retains
its separate explicit opt-in. A full live-provider and packaged end-to-end release
rehearsal remains required before launch claims.
