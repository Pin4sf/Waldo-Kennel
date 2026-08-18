# Kennel documentation

Kennel currently consists of an AO-derived Go daemon and Electron supervisor plus a prototype Outcome UI. The documents below describe that foundation. Mission, personal memory, Waldo authority/verification semantics, and Xirp/Medley/Paxel integrations are not shipped.

## Foundation first

| Document | Purpose |
| --- | --- |
| [Foundation acceptance — 2026-08-18](foundation-acceptance-2026-08-18.md) | F0-F6 scope, evidence, exclusions, dev-dependency debt, and the next architecture boundary. |
| [Identity and state](identity-and-state.md) | Kennel-owned bundle, executable, protocol, updater, state, environment, port, branch, and renderer identities. |
| [Upstream provenance](upstream-provenance.md) | Pinned AO source, license, non-destructive sync seam, and the separately approved ancestry-repair plan. |
| [Status](STATUS.md) | Current chassis capabilities, donor surfaces, and work that is not shipped. |
| [Development](development.md) | Reproducible bootstrap and verification commands. |

## Chassis references

These documents describe the inherited and currently working orchestration implementation. AO vocabulary can remain where it identifies a source compatibility seam; it is not Kennel's installed identity.

| Document | Purpose |
| --- | --- |
| [Architecture](architecture.md) | Backend model, persistence/CDC, lifecycle, API, terminal, browser, and load-bearing boundaries. |
| [Backend code structure](backend-code-structure.md) | Go package ownership and dependency rules. |
| [CLI](cli/README.md) | The `kennel` thin client over daemon HTTP. |
| [Stack](stack.md) | Accepted and pending runtime/library choices inherited by the chassis. |
| [Telemetry](telemetry.md) | Current optional telemetry behavior and privacy safeguards. |
| [Cloud development](cloud-development.md) | Retained optional cloud donor/compatibility work; not a deployed Kennel cloud claim. |
| [Cloud refactor](cloud-refactor.md) | Existing shared contracts and donor boundaries. |

## Architectural rule

Persist durable facts and derive display status. Session activity, termination, PR/check/review facts, and change-log events are durable; a display label is not. Product architecture added after the foundation must preserve truthful, inspectable evidence and explicit human authority.
