# Outcome deletion verification

Worktree: `/private/tmp/kennel-launch-candidate-20260911`.
Base for this slice: `81fd708e2`.

Implements recoverable Trash/Restore and explicit permanent deletion. See [ADR 0014](../adr/0014-owner-directed-outcome-erasure.md) for ownership and cleanup boundaries. The dialog shows the Outcome title and requires the word `confirm`; counts and workspace paths are collapsed under “What will be removed.” No existing user Outcome has been erased.

## Automated evidence

- Real SQLite tests: Trash/restore/permanent round trip; stale revision refused; unrelated Outcome/project preserved; active Attempt and custody block; owned terminated session purges; audit payload preserved while its deleted-session FK is detached; normal immutable Contract DELETE still rejected; late document/delivery insertion fenced; transaction authority cleared.
- Retained filesystem tests: removing an owned symlink does not traverse it; path traversal and replaced document root refused.
- Service tests: invalid confirmation, stale revision and active execution do not mutate state; explicit confirmation follows Trash and durable cleanup admission before purge.
- Renderer tests: visible title, both choices, explicit confirmation word, active blockers, pending cleanup prevents Restore.
- Full Go build/vet/test passed. Affected SQLite/service/artifact/HTTP suites and HTTP/spec parity passed. The final full `go test -race ./...` passed on `5a75940d3`, including SQLite store (313.526 seconds).
- Full frontend suite: 227 files, 2741 passed, 6 skipped. Typecheck passed. Localized deletion/i18n checks passed. Earlier execution without localhost permissions failed socket tests; the subsequent permitted run passed.
- Go lint: zero findings after resolving integrated-branch findings as well as deletion changes.
- Final packaged Electron build passed. Bootstrap passed. The complete `npm run test:foundation` gate passed on `5a75940d3`: Go build/test/vet, shared packages, Island, frontend typecheck/test/build, pod gate and clean sqlc/API regeneration. `npx @redwoodjs/agent-ci run --all` exited 0 but reported no relevant workflows for this branch; it adds no test evidence.

## Real daemon and packaged app

The rebuilt package launched against the existing isolated audit profile after a database backup. Disposable fixture `out-ae30fee9-f395-40ec-a69f-e157a3e20a00`, titled “Deletion verification — disposable,” contained a Contract but no Attempt, session or workspace.

- Native UI: Move to Trash succeeded and the daemon Trash list confirmed it. The original agent-orchestrator Outcome's deletion preview displayed its unresolved execution blocker with both destructive actions disabled.
- Final packaged daemon: Trash, Restore, incorrect-confirmation rejection (400), and permanent deletion (`confirm`) passed. Subsequent preview returned 404. Both existing user Outcomes remained accessible. This was an HTTP integration test, not a native permanent-delete click journey.
- Final native retest was incomplete: accessibility navigation updated, but screenshots and click responses intermittently remained stale. The final title/confirmation copy is covered by renderer tests; the complete final native journey is not claimed.
- The fixture had no runtime resources. Actual provider termination and real workspace cleanup are not established by this fixture; service/filesystem/storage tests cover their boundaries.

## Review and scope

Independent source review found shared Project conversation ownership, late document/delivery admission, and workspace-free cleanup issues. They were repaired and re-reviewed with no remaining merge blocker reported. This is source review, not additional live execution evidence.

An additional proposal to purge audit entries by JSON identity matches was rejected by automatic approval review as too broad. It was not applied. Audit payloads, backups and logs remain; the exact owned session FK is detached to allow session-row erasure. Imported projects and exported files remain outside cleanup.

This feature does not resolve native Codex reasoning confinement, Claude resume conformance, OpenCode conformance, or the existing unresolved agent-orchestrator Attempt. It does not constitute launch or owner acceptance.
