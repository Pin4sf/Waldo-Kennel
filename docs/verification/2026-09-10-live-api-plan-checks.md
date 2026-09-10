# Real API planning checkpoint

Target: local integration branch codex/mission-api-integration, base ac3e08fcc. No push, merge to beta, execution or owner Outcome acceptance.

## Observations

- OpenAI gpt-5-nano synthetic connectivity request completed (72 tokens).
- Isolated real daemon: Project registration 201; intake capture 201; live grounded proposal 200 (~15 seconds); Contract confirmation 201; Plan generation 201 (~23 seconds).
- Original Plan had no approvedChecks. Root cause: planReply parsed checkCommands, but planSchema omitted that property while forbidding additional properties. Added the schema and prompt guidance; daemon check compiler remains authoritative.
- First live replan after schema repair rejected C0 as unknown (HTTP400 PLAN_DRAFT_CRITERION_UNKNOWN). Constrained criterion aliases in the model schema to the actual Contract aliases.
- Next live replan returned 201 (~4 seconds) with a criterion-bound check. It is NOT approved: `python3 greet.py` exits zero for either greeting, so it does not prove the output criterion. The proposed unit requests read/execute but omits write despite modifying the file. Model also lists ordinary constraints as blockers.
- Proposed Plan response createdAt remains zero; separate serialization/storage follow-up.

## Verification and boundaries

Regression initially failed because checkCommands was absent. Tests now cover schema availability and preserving argument boundaries/newlines, criterion and timeout through LLM response decoding. Focused intelligence/outcome suites and daemon build pass. Full backend test log: /tmp/kennel-check-schema-full-tests.log (final result recorded separately).

Synthetic data only; no user repository contents sent. Key read privately from local .env, never put in source/logs. Root .env ignore rule added in primary checkout separately. Tests used daemon HTTP, not Electron interaction. Disposable source and records: /var/folders/fs/sqsvy0pn0n72nwp8px8pwg_h0000gn/T/kennel-api-flow-hwl3u1xq/ (result.json, plan-result.json, replan-result.json). Test daemon stopped after each request sequence.

## Next acceptance gate

Request correction with clear feedback: modification requires write intent; verification must fail against the unchanged baseline and pass against the desired result; constraints are not unresolved blockers. Inspect the new proposal before approval. Never silently change the model's authority or accept a check merely because it exits zero. General arbitrary-command semantic verification is not solved by this schema fix.

Pending product work remains: real execution, independent criterion evidence, restart/continuation, rework, durable delivery integration, worker bridge, Outcome conversation, usage and native Electron journey. Codex harness reasoning remains unavailable due unproven confinement; direct API mode is explicit.
