# Launch usability iteration

Worktree: `/private/tmp/kennel-launch-candidate-20260911`, based on `92aa0f14b`.

## Changed behavior

- Four board buckets: To do, In progress, Needs you, Finished. Only accepted Outcomes enter Finished. Reviewable and unavailable Outcomes remain in Needs you. Board and List share filtering; the default includes finished work.
- Outcome details scroll independently beneath their header. Contract sections collapse individually and use a bounded content width.
- Focus Outcome and Show board beside Outcome operate on the named Mission container. Narrow windows hide the split control.
- Permissions are editable before confirmation. Cancel request replaces release-intake terminology, with cancellation tucked into a disclosure.
- Failed Plan drafting has the correct heading, a readable explanation, and the daemon's specific reason.
- An absent asynchronous analysis request no longer falsely means offline generation. The UI explicitly says when authorship was not recorded.

Existing locale catalogs receive English wording for changed keys; this is consistent terminology, not completed translation review.

## Verification

Frontend typecheck passes. Full frontend suite: 225 files, 2736 passed, 6 skipped. Read-only Spec and Standards reviews identified filter inconsistencies and an overly specific error explanation; corrected, with filter regression coverage.

Real packaged desktop with isolated audit data on port 65025: scrolling reached Permissions and supplied documents; disclosure sections collapsed; maximizing exposed the split, Focus Outcome removed the board, and Show board beside Outcome restored it. A real API-generated proposal was created for the imported agent-orchestrator project and its Network and Pull request permissions were switched off in the UI.

Codex 0.153.4: the existing live adapter test passed creation, streamed response, and reopening the same provider thread in a fresh process. This does not prove Outcome reasoning, governed execution, interruption, proof, or delivery.

## Still open

- GPT-5 nano reasoning is verified against the configured API credential, but repeated live Plan drafting still produced a shell check rejected by `PLAN_DRAFT_CHECK_INVALID`. The more explicit planner prompt did not resolve this; validation remains intact.
- Claude's packaged ACP live test passed initial conversation but failed changed standing-context behavior on resume: the answer repeated the earlier role. Root cause is not yet isolated between adapter, runtime, replay and model behavior.
- OpenCode is absent from the current shell PATH; no live OpenCode session is claimed.
- Codex bounded reasoning remains disabled. A credential-free local API inventory probe using empty environments, disabled shell/browser/apps/plugins, empty capability roots and disabled host skill discovery still exposed `request_user_input` and `skills`. It is not yet a proven no-tool boundary. Account/configuration-bound verification remains required too.
- Permission editing after Outcome confirmation is not exposed by this slice. A safe revision UI must preserve the full Contract, not silently discard evidence expectations, facets or temporal fields absent from the current revision DTO.
- Full real-project execution, proof, delivery and owner acceptance remain unverified. This checkpoint is not launch acceptance.

Logs are under `/private/tmp/launch-ux-checkpoint-*`, `/private/tmp/launch-codex-session-live.log`, `/private/tmp/launch-claude-session-live.log` and `/private/tmp/kennel-codex-tool-inventory.json`. The active isolated audit profile is `/Users/shivanshfulper/.kennel/ux-audits/20260911T094417Z-launch-candidate-20260911-29c85768`.
