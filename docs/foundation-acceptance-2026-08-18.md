# Foundation acceptance record — 2026-08-18

This record defines the F0-F6 gate that must be green before Waldo product architecture or feature implementation begins. Evidence applies to branch `codex/foundation-gate-f0-f6` based on Kennel `main` at `b1bcd37fd0911a86da5bb21c07ac3a0b9adecb3b`.

## Scope and exclusions

This gate covers repository provenance, Kennel identity and state isolation, an inherited migration collision, minimum CI/security controls, deterministic bootstrap and dependency policy, and truthful documentation.

It does not implement Mission, personal memory, a personal-agent model, Waldo authority/verification semantics, Xirp, Medley, Paxel, or another product feature. It does not merge, deploy, publish, or launch the desktop application. The prototype Outcome UI and AO-derived orchestration features already in the source tree are preserved without being promoted to claims about the future product.

## Gate evidence

| Gate | Result | Evidence |
| --- | --- | --- |
| F0 repository baseline | Complete | Clean `main` matched `origin/main` at `b1bcd37f`; work moved to `codex/foundation-gate-f0-f6`. The public repository used `main`; live protection/security settings are recorded in the PR handoff. |
| F1 provenance and sync | Complete | [`upstream.json`](../upstream.json), [`NOTICE`](../NOTICE), [`upstream-provenance.md`](upstream-provenance.md), and `scripts/compare-upstream.sh` pin AO SHA `66ba38b0` / tree `d06dd98b`. The audited trees share 2,406 exact same-path blobs across 2,468 shared paths and have no merge base. |
| F2 identity and state | Complete | [`identity-and-state.md`](identity-and-state.md), runtime tests, and `frontend/scripts/assert-package-identity.mjs` define and assert Kennel-owned bundle, executable, protocol, updater, state, environment, port, release, branch, and renderer identifiers. No application launch was used. |
| F3 migration repair | Complete | `backend/internal/storage/sqlite/db.go` adds idempotent physical-schema reconciliation without editing migration 0098. `migrate_project_chat_projection_test.go` upgrades an AO-derived fixture that already burned the conflicting version and proves both schemas are present. |
| F4 CI and security | Complete in repository | `.github/workflows/ci.yml` runs bootstrap, production audits, foundation checks, lint, race tests, and a macOS package-identity job. `.github/workflows/security.yml` runs Gitleaks, govulncheck, and production audits. Third-party actions are pinned to immutable SHAs; Dependabot is configured. Live repository controls are applied only through normal GitHub settings, never history rewriting. |
| F5 bootstrap, tests, dependencies | Complete | `.nvmrc`, root engine/package-manager fields, `scripts/bootstrap.sh`, `scripts/test-foundation.sh`, and `scripts/audit-production.sh` normalize all active package sets. The local foundation matrix and all production audits passed. |
| F6 truthful documentation | Complete | README and docs distinguish current chassis, prototype/donor surfaces, deliberate source compatibility seams, and unimplemented Waldo product architecture. |

## Verified local matrix

The foundation run completed these checks from a fresh normalized install:

- Go build, full tests, vet, generated API/sqlc drift checks, and migration upgrade tests.
- Cloud client: 1 test file / 21 tests.
- Product UI: 10 test files / 59 tests and production build.
- Desktop frontend: 192 test files, 2,490 passed tests and 6 intentional skips, plus typecheck/build coverage in the gate.
- Retained landing donor: production build and 46 generated Markdown twins.
- Repository scripts: 17 Node tests.
- Production `npm audit` for root, product UI, cloud client, ACP runtime, desktop, and landing: zero known vulnerabilities.
- macOS package inspection: `Kennel.app`, `in.heywaldo.kennel`, executable `kennel`, protocol `kennel-app`, updater cache `kennel-updater`, and release target `Pin4sf/Waldo-Kennel`.

The full development dependency audit is not clean because inherited build/test toolchains remain: root 3 advisories (1 moderate, 2 high), cloud 2 high, desktop 36 (3 low, 1 moderate, 31 high, 1 critical), and scripts 4 (1 moderate, 3 high). Product UI, ACP runtime, and landing have zero. These advisories are explicit dev-toolchain debt; the foundation gate fails on shipped production dependency vulnerabilities and lets the scheduled security workflow track the broader debt.

## Preserved donor and compatibility surfaces

- `frontend/src/landing`: retained AO marketing donor; compiled in the foundation matrix but not the desktop launch surface or a Kennel release claim.
- `packages/mobile`: retained Expo donor; not part of the current shipped Kennel claim.
- `packages/cloud-client`: retained compatibility package and tested foundation, not proof of a deployed cloud product.
- `packages/ao*`: frozen legacy AO packages, not renamed or republished.
- release, pod, and updater scripts: retained for controlled migration; no release was published by this gate.
- `backend/cmd/ao`, Go module imports, `.ao/attachments`, and `.ao/launch.json`: documented source/project compatibility seams, not installed identity.

## Known governance boundary

Kennel was published as a manual/squashed copy and has no shared Git ancestry with AO. Faithful ancestry repair would replace all published Kennel commit IDs and force-update `main`. The verified non-destructive synchronization seam is active; ancestry repair is a separate maintainer-approved migration with a freeze, backups, tree-equivalence proof, and rollback ref. It was not performed here.

## Next architecture entrypoint

After this gate is reviewed and accepted, define the product architecture for Mission and the Waldo personal agent before writing feature code. That decision must specify user ownership, authority, memory admission/correction/release, verification, human acceptance, and how Kennel remains one presence of Waldo. Existing Outcome and AO orchestration concepts are inputs, not a locked product model.
