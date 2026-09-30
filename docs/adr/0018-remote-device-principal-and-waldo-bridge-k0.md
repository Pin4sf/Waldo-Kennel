# ADR 0018: Remote device principal and the Waldo device bridge (K0)

- Status: proposed; owner review pending
- Date: 2026-09-27
- Scope: Kennel-side authority and transport model for the Waldo device bridge, K0 milestone (pairing + first round trip)
- Companion contract: [Waldo device bridge contract v0.2.3](../contracts/waldo-device-bridge-v0.2.3.md)
- Builds on: [harness connection and authority](../architecture/harness-connection-and-authority.md), [local owner command authority](../architecture/local-owner-command-authority.md), ADR 0017

## Context

The owner has set the product pivot: Waldo is the personal entry point and memory home; Kennel is the Mac execution and machine-eyes layer (owner-requested pivot brief, 2026-09-26). The first bridge milestone is K0, defined by the owner on 2026-09-27 as **pairing + first round trip** per the Waldo build lane's bridge docs (`WALDO_KENNEL_BRIDGE_2026-09-24.md` K1/B1 + K2/B2, restated as "K0" in `WALDO_KENNEL_BRIDGE_UPDATE_2026-09-26.md` §5). Capture, catalog/sync, and governed mission tasks are later milestones and are out of scope here.

Kennel's frozen authority model today:

- Owner commands are accepted only on the daemon's loopback listener with a constant-time bearer check; a standalone daemon mounts no owner-command route, fail-closed by design ([local owner command authority](../architecture/local-owner-command-authority.md)).
- Pairing exists only as harness-adapter pairing: challenge fields include InstallationID, AdapterDigest, HarnessIdentity, ProviderVersion, ProtocolFingerprint, MissionID, AppRunID; kinds are `pair`/`rotate`; issuance requires an already-authenticated internal S1 caller (`backend/internal/harnesspairing/coordinator.go`). An adapter cannot mint an owner principal ([harness connection and authority](../architecture/harness-connection-and-authority.md)).
- Owner proof is one-time, bound to content digest, target digest, command class, mission, app run, and expiry; material classes additionally require a native ConfirmationRef (`backend/internal/ownerproof/kernel.go`, `backend/internal/domain/owner_proof.go`).
- No cloud-to-daemon relay exists anywhere in the backend. The only WebSocket client in the repository is `@aoagents/cloud-client`, a deprecated AO dependency; this ADR forbids building on it.
- The command vocabulary is session-scoped (turn/steer/answer/interrupt/cancel/replace/approval/accept). Nothing covers a remote device, machine-state query, or local notification.

K0 therefore requires a new principal kind, a new outbound transport surface, and a small device-command vocabulary, added without weakening any frozen rule above.

## Decision

### 1. A new principal kind: the paired device

Kennel adds a **device principal**, distinct from the local-owner principal and from harness-adapter connections. A device is a remote, owner-bound endpoint reached only over the bridge. Pairing follows the one-time-code pattern the owner already approved for the bridge design: the owner requests a pairing code from Waldo; Kennel redeems it; Kennel generates an ed25519 device keypair locally and the private key never leaves the Mac; the backend stores the device row (owner, public key, declared capability classes, label, created/last-seen). Revocation deletes the device row; the daemon must observe rejection and present a truthful unpaired state.

Pairing authenticates transport only. It never proves that content came from the owner, never mints or widens owner authority, and never substitutes for owner proof. This restates the frozen harness rule for the new principal.

### 2. Outbound-only transport

The bridge is a device-initiated outbound WebSocket connection from the daemon to the Waldo backend. Local listeners remain loopback; no new inbound network surface is created. Every device request is signed (ed25519 over timestamp, method, path, body hash) with a replay window enforced by timestamp and nonce. Heartbeat and capability advertisement run on every connect. While disconnected, outbound messages spool in a durable on-device outbox and drain on reconnect without duplicates; state shown to the owner is honest (paired/unpaired, online/offline), never optimistic.

### 3. Device-command vocabulary, K0 subset

A new frozen vocabulary, separate from the session-scoped owner-command classes, versioned as the bridge contract (v0.1). K0 admits exactly two command classes:

- `machine_state_query` - a read-only query answered only from state the daemon actually holds (session/Attempt state, liveness, workspacewatch facts). No inferred answers, no MissionProjection dependency.
- `notify_local` - a notification surfaced on the Mac through the island or native notification, with delivery state returned.

Plus the protocol messages `ack`, `result`, and `receipt`, with monotonic revisions and idempotency keys. Repeated delivery of a command maps to one effect; a reconnect reconciles in-flight work before issuing anything new.

K0 commands carry no owner authority and cause no local effects beyond a notification. No owner proof is exercised in K0.

### 4. Transitional authority rule and its destination

For K0, material authority transitions (Contract/Plan approval or revision, capability widening, external effects, Attempt replacement, Accept) remain native on the Mac, exactly as frozen today. The bridge carries no approval path. The owner approved this rule on 2026-09-27 as **transitional** (WhatsApp, 11:16 AM): the destination is Waldo-handled approval without Mac-side acceptance.

That destination is gated on a designed remote owner-proof mechanism: owner proof minted off-Mac by a key the owner holds (phone-key/passkey-class), bound to the same exact content/target digests, command class, mission, app run, and expiry as today's proof, and verified by the daemon with the same one-time, constant-time, fail-closed semantics. Until that mechanism is designed, reviewed, and accepted as its own ADR amendment, no policy wording, channel message, or bridge content relaxes the native rule. A WhatsApp reply is not owner proof.

### 5. New code, not AO reuse

The bridge client, device pairing coordinator, and durable bridge state are new Go code in the daemon. `@aoagents/cloud-client` is not used, extended, or wrapped.

### 6. Documentation and status

This ADR and the companion contract are proposed together. The contract version both sides implement is the one the owner accepts; the Waldo build lane signs the same version before Slice 3's live check. Kennel STATUS.md and the execution map gain one line each naming the bridge as a parallel track that does not displace the Stage 4-11 exits.

## Consequences

- A second pairing subsystem exists beside harness pairing. They share the one-time-challenge pattern but no state, no store tables, and no authority. The device coordinator never calls the harness connection kernel.
- The daemon gains one outbound network client and a durable outbox/journal (new SQLite tables). The CDC poller's in-memory cursor is unaffected; the bridge outbox owns its own durable cursor.
- `machine_state_query` answers are only as good as existing daemon state until MissionProjection (K3 milestone, also P0 gap #3) lands. The contract says so; the UI must not dress a partial answer as complete.
- Revocation UX becomes a K0 deliverable: truthful paired/unpaired state locally, and a backend console devices page (backend lane).
- The remote owner-proof design becomes the named gate on every later milestone that carries owner authority over the bridge (K3 approvals, K6 mission tasks). It is scheduled work, not a loophole.
- Multi-device is supported by the model (one owner, n device rows) even though beta ships one Mac.

## Acceptance falsifiers

- A paired device can approve, widen, or accept anything, or cause a local effect beyond a notification, without separate exact owner proof minted under the frozen rules.
- Pairing content is treated as owner content anywhere in the daemon.
- An unpaired, revoked, expired, or signature-invalid device frame reaches command processing or state mutation.
- A replayed command (same idempotency key) causes a second effect; a replayed frame with a different fingerprint is accepted as the original.
- A daemon restart loses pairing state, spooled outbound messages, or inbound command journal entries; restart causes a duplicate effect on recovery.
- The bridge dials or accepts any inbound listener, or the daemon exposes a non-loopback route as part of K0.
- The UI shows paired/online before a real signed handshake and capability exchange complete, or shows optimistic state while the outbox holds undelivered results.
- `machine_state_query` returns state the daemon does not durably hold, or silently depends on MissionProjection.
- Any K0 code path imports or shells out to `@aoagents/cloud-client`.
- A channel message (WhatsApp or other) is accepted as owner proof for any class. 