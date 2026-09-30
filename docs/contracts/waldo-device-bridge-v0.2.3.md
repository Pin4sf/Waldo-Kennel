# Waldo device bridge contract v0.2.3

- Status: **presented for owner sign-off.** On owner and backend-lane sign-off, v0.2.3 supersedes owner-side accepted v0.2.1 and the unsigned v0.2.2 proposal. This is the first text presented for both lanes to sign; the Waldo build lane must sign this same version before any live check. Historical owner-side acceptance of v0.1 (2026-09-27 7:43 PM IST), v0.2 (2026-09-28 1:23 AM IST), and v0.2.1 (2026-09-29) did not constitute joint sign-off; the backend lane declined to sign v0.2.1 pending the corrections below.
- Date: 2026-09-29
- Scope: K0 milestone - device pairing + first round trip (`machine_state_query`, `notify_local`). Later milestones (mission projection, capture, catalog/sync, mission tasks) amend this contract by version, never by silent drift.
- Governing ADR: [ADR 0018](../adr/0018-remote-device-principal-and-waldo-bridge-k0.md). ADR 0018 §3 admits exactly two command classes for K0; v0.2.3's taxonomy (§5) matches it. The ADR's "(v0.1)" mention names the versioning mechanism, not a pinned number; no ADR change is required.
- Backend-side facts cited from `waldo-backend` `beta-mvp` docs (`WALDO_KENNEL_BRIDGE_2026-09-24.md`, `KENNEL_BRIDGE_BACKEND_2026-09-24.md`, `WALDO_KENNEL_BRIDGE_UPDATE_2026-09-26.md`) are unverified at source. Where backend behavior differs, this contract governs only after both sides agree in writing.

## Final v0.2.3 retention erratum (Ashish's 2026-09-29 11:39:52 PM IST ruling; backend-lane acceptance pending)

Ashish chose Option A (his WhatsApp reply: "A") after a choice between keeping a small backend result-dedup record for the Mac's entire paired lifetime and adding another receipt-confirmation wire step. This is the last open design choice reported by the backend lane at this review; their acceptance of this exact amended text and joint signature are still pending. §6 retains the minimal dedup fingerprint and receipt metadata while the device may authenticate and retransmit, without treating receipt issuance, socket delivery, or aggregate heartbeat depth as proof the device applied a receipt. The five existing protocol types, envelope and signing vectors are unchanged.

## Focused v0.2.3 errata (owner-approved notification choice; backend-lane acceptance pending)

1. **Error direction (§5.3):** A well-formed backend command rejected after authenticated device dispatch gets a device-origin `ack` with a typed reason. Unattributable malformed frames and invalid device-origin results/acks/heartbeats, and a backend-origin receipt naming an unknown result, close the socket with no typed wire response and no effect; server-side diagnostics may retain typed reasons. No new protocol type.
2. **Backend persistence (§6):** A duplicate matching result gets a new backend receipt after restart or lost receipt; backend dedup/result metadata persists as long as the device can authenticate and retransmit its outbox, including beyond command expiry and replay horizon, under the final retention erratum above. A mismatch never gets a receipt. The device can drain its outbox without waiting for a volatile ack.
3. **Pairing (§2):** Mint at least 256 CSPRNG bits as 43-character unpadded base64url, enforce a 600-second expiry, single-use atomic hash reservation, and a bounded deploy-profile rate budget. The test vector uses a syntactically valid fixed code; it is not a secret.
4. **Notification issuance (§7):** Ashish chose Option A on 2026-09-29 at 11:34:36 PM IST ("go with A , we can’t have it every single time."): a separately authenticated owner request per notification OR an exact, revocable standing grant, with backend scope check, neutral nonsensitive display text, and per-device limits of 5/60 seconds and 20/hour. This owner ruling permits proposing that policy to the backend lane; it does not itself create any standing grant or authorize an actual notification. Pairing alone remains insufficient. Backend-lane acceptance and final joint signature are pending.

## Errata-2 on proposed v0.2.2 (historical owner rulings carried into v0.2.3) (Ashish's 2026-09-29 11:23 PM IST rulings; backend-lane acceptance pending)

1. **Device ID:** Ashish chose backend-assigned `device_id` at successful redeem, retaining the five-field signed request. The Mac generates its keypair before redeem and stores the returned ID with that keypair; it does not supply an ID in the request (§2.2). This resolves the Mac agent's conflicting earlier account of a device-generated ID; no device-ID field is added to the redeem golden vectors.
2. **Device-scoped storage:** Ashish chose one forward migration of the already-shipped 0162 bridge tables, not parallel active v0.2.2 tables. The journal and outbox identities become device-scoped while existing rows are preserved or the migration fails atomically (§6). This is a Kennel-side implementation decision, not a new wire field. The next free migration number must be confirmed across live branches, ledger and other work (including K4) before allocation; do not assume 0164 is free.
3. **Query errors:** Ashish chose a `failed` result for genuine processing failure, with `reason:"processing_failed"` and no `answer`. Missing durable evidence instead yields an `answered` result whose answer state is `unknown` (§§5.2-5.3).

These are owner rulings on the three open choices, not backend-lane acceptance or joint signature. The v0.2.3 notification-policy choice is separately recorded above; no other v0.2.2 review choice is silently treated as a joint agreement.

## Changelog vs v0.2.1 (historical v0.2.2 proposal)

1. **Wire shapes incorporated:** closed per-type schemas, payload placement, unambiguous receipt reference, bounded fields, query answer schemas, error taxonomy and connect capability declaration (§§2-5).
2. **Canonicalization incorporated:** strict UTF-8 and duplicate-key rejection before signature verification, canonical wire bytes, and independently reproducible signature vectors (Appendix A).
3. **Atomic lifecycle incorporated:** raw-body redeem key proof, one-winner code redemption, generic throttling, atomic nonce admission, and revoke cleanup (§§2-3).
4. **Reconciliation narrowed:** durable device-scoped message fingerprints, atomic result/receipt state changes, bounded tombstones and expiry behavior (§6). At that stage, notification display was bounded text with no authority-bearing command. v0.2.3 proposes a separate owner-source and display policy (§7); this is the pending negotiation point.
5. **Trust/error controls incorporated:** generic pre-auth failure bytes versus post-auth typed errors; fixed HTTPS origin, redirects forbidden and certificate failure fatal (§§3, 5.3).

## Changelog vs v0.2 (v0.2.1 consolidation)

1. Backend-to-device frames use TLS server authentication, not message-layer signatures; the device signs every frame it sends. Symmetric backend frame signing is a candidate for a later version bump and needs owner approval.
2. Redeem and connect signatures now bind the HTTP nonce: `timestamp LF nonce LF method LF path LF body_sha256`.
3. A re-sent result keeps its logical identity and payload, but uses a fresh timestamp and nonce and is re-signed; receivers dedup by `message_id`, not signature bytes.
4. Canonical JSON, clock sync, and redeem public-key binding are pinned in §§2-3.1. This version incorporates the corrected historical changelog and owner-side, not joint, acceptance history below.

## Changelog vs v0.1 (historical v0.2 precision changes, corrected in v0.2.1)

1. **§3 signing is now exact.** Canonical base-string layout per surface, timestamp encoding (Unix seconds, decimal ASCII), hash encoding (lowercase hex SHA-256), signature encoding (base64url, no padding), nonce format, replay window values, and the exact header/field names where signature and nonce appear on redeem, connect, and every socket frame.
2. **§4/§5 revision semantics settled.** A command has exactly ONE revision. `ack`, `result`, and `receipt` are distinct protocol message types that REFERENCE `(command_id, revision)`; they are not revisions of the command. This matches the Kennel Slice 2 store as built on the Kennel side: exactly one durable outbox row per `(command_id, revision)` (migration 0162), holding the result.
3. **§5 taxonomy corrected.** v0.1 listed five "frozen classes," mixing commands with protocol traffic and contradicting its own heartbeat requirement (and ADR 0018's "exactly two command classes"). v0.2 separates **protocol message types** (`command`, `ack`, `result`, `receipt`, `heartbeat` - transport-level, present in every version) from **command classes** (`machine_state_query`, `notify_local` - the frozen K0 vocabulary). The envelope gains a `type` field; only `type: command` carries `class`.
4. **§4 envelope** gained `type`, `timestamp`, `nonce`, `signature` in v0.2; the `revision` rule then said exactly one per command. The v0.2.1 consolidation declared `contract_version` as `0.2.1`; v0.2.2 proposed `0.2.2`, and this proposal declares `0.2.3`.
5. Section 6 retransmission semantics are NEW, not carried from v0.1: same-`message_id` result re-send (freshly re-signed per §3.1; dedup by `message_id`, never by signature bytes), receiver dedup by `message_id`, and the re-ack-and-continue rule. v0.1's §6 defined ordering and replay but no retransmission rules - the retransmission blocker sat there. Everything else - identities, pairing lifecycle, transport shape, offline behavior, ordering and replay beyond the new retransmission rules, authority and privacy invariants, versioning rule, K0 exit criteria - is v0.1 text, unchanged in meaning.
6. The canonical base-string separator is pinned as LF (0x0A), settling v0.1's ambiguous displayed "|" separator. No implementation was built on v0.1, so this is a legitimate versioned clarification, not a wire break.

## 1. Identities

- **Owner**: the Waldo account holder. Bound to devices 1:n.
- **Device**: one paired machine. Identity = an ed25519 keypair generated on the device at redeem time. The private key never leaves the device, never enters prompts, logs, argv, repository files, or any message defined here.
- **Device principal (Kennel-side)**: the daemon's representation of the paired remote endpoint. It is not an owner principal and not a harness connection (ADR 0018 §1).
- **Daemon**: the Kennel Go daemon on the device. **Backend**: the Waldo backend device service.

## 2. Pairing protocol

1. **Mint (backend, owner session)**: owner requests pairing from any Waldo channel or console. Backend mints exactly 32 bytes with a CSPRNG and presents the unpadded base64url encoding (43 characters, 256 bits of entropy) as a one-time pairing code. The code expires at `created_at + 600` seconds, with expiry checked against server time during redemption; it is single-use and stored only as its SHA-256 digest at rest. The digest has a unique index; a digest collision fails minting and triggers a fresh draw, never an overwrite. Mint and redeem tests must prove unique-code reservation, one winner under parallel redemption and expiry at the exact 600-second boundary.
2. **Redeem (device -> backend)**: `POST /devices/redeem` with JSON body `{ code, device_pubkey, label, declared_capabilities, contract_version }`, signed per §3.1 (headers). `device_pubkey` is the raw 32-byte ed25519 public key, base64url without padding (43 ASCII characters). `code` is exactly 43 unpadded base64url ASCII characters decoding to 32 bytes; `label` is 1-120 UTF-8 bytes; the redeem raw body is at most 2048 bytes, strict UTF-8 JSON with unique keys and exactly the five listed fields. `declared_capabilities` is a JSON array of distinct command-class strings from §5.2 in the order `machine_state_query`, `notify_local` if both are present (nonempty, no duplicates). `contract_version` is `0.2.3`. On successful atomic redemption the backend generates `device_id`, stores the device row (owner, device ID, pubkey, capabilities, label, created_at, last_seen_at), and returns `{ device_id, owner_id, accepted_contract_version }`, with `accepted_contract_version` set to `0.2.3`. The device persists the returned backend-assigned ID with its locally generated keypair; it must not supply `device_id` in the five-field redeem body. No device identity is paired until redemption succeeds. The `device_pubkey` is self-asserted at redeem; what binds it to the owner is the one-time pairing code (single-use, short-lived, stored as SHA-256 at rest). After a successful redeem the backend trusts exactly this key for this device row; a later key can only arrive through a new pairing code, never by update. A used, expired, or unknown code fails with the same generic pre-auth error bytes (§5.3). Before attempting redemption, parse the raw JSON body strictly, extract `device_pubkey` from that SAME raw body, decode exactly 32 ed25519 public-key bytes, and verify `X-Waldo-Signature` over the raw-body hash (§3.1) with that key; do not trust a separately supplied or reconstructed key. In one database transaction, atomically mark the still-valid pairing-code hash used and insert exactly one device registration; a unique constraint on the code hash plus conditional first-wins update makes racing redeems produce one row and one winner. The loser receives the same generic pre-auth error. Rate-limit before code lookup using a deployment profile fixed and tested by both lanes before live K0: no more than 5 redeem attempts per source IP per 60 seconds and 20 per source IP per hour, and no more than 5 attempts per presented code digest per 600-second code lifetime, aggregated across backend workers and restarts. Excess attempts receive the identical generic pre-auth response, with no code-existence signal. Changing these ceilings requires a reviewed deployment-profile change; brute-force safety is an acceptance test, not merely a recommendation.
3. **Revoke**: backend deletes the device row (owner action from console or channel). The backend immediately closes that device's active socket, atomically cancels all queued undelivered commands for it, and rejects later device authentication/signatures. The daemon observes rejection, clears its paired state locally (keeping the audit journal), and shows unpaired. Re-pairing mints a new device identity; it never resurrects a revoked row.
4. **Rotation**: not in v0.2.3. A lost key means revoke + re-pair.

## 3. Transport and signing (exact)

Device-initiated outbound WSS only: `GET /devices/connect` (upgrade), device-signed. The device uses one configured, pinned HTTPS origin (scheme, host and port, not a certificate/public-key pin) for redeem and connect, follows no HTTP redirects, verifies the certificate chain and hostname, and aborts/closes on any certificate verification failure. TLS server authentication is the sole backend-to-device authenticity control under Option A. The connect request declares `contract_version=0.2.3` and `declared_capabilities=machine_state_query,notify_local` as exact query parameters (a subset is a comma-joined list in §5.2 order, no spaces; percent-encoding is forbidden for these ASCII values). The signed `path` for connect is the exact path AND query, `/devices/connect?contract_version=0.2.3&declared_capabilities=machine_state_query,notify_local` (or the subset string), in that order (§3.1); this binds capability claims to the signature. Redeem signs `/devices/redeem`. The backend requires these values to match the paired device row and rejects unsupported declarations. No other connect query parameters are allowed. The daemon opens no inbound listener for the bridge; local routes stay loopback.

### 3.1 Canonical signature base string

Every signed surface uses the same construction: fields joined by a single LF byte (`0x0A`), UTF-8, no trailing LF, no escaping. Missing body means the empty-body hash below.

- **HTTP surfaces** (redeem `POST /devices/redeem`, connect `GET /devices/connect`):
  `timestamp LF nonce LF method LF path LF body_sha256`
- **Socket frames from the device** (every frame after upgrade):
  `timestamp LF type LF message_id LF nonce LF body_sha256`

Field encodings:

| Field | Encoding |
|---|---|
| `timestamp` | Unix seconds, decimal ASCII, no leading zeros (example: `1790200800`); max 10 digits |
| `method` | uppercase HTTP method (`POST`, `GET`) |
| `path` | exact request target: `/devices/redeem` for redeem; for connect, `/devices/connect?contract_version=0.2.3&declared_capabilities=` followed by the comma-joined capability subset in §5.2 order. No other spelling, encoding or query fields are accepted |
| `type` | the protocol message type from §5.1 |
| `message_id` | the envelope's `message_id`, verbatim |
| `nonce` | see below |
| `body_sha256` | lowercase hex (64 chars) of SHA-256 over the exact body bytes. For HTTP: the raw request body. For frames: the exact received canonical UTF-8 JSON bytes of the full envelope **excluding** the `signature` field (the sender serializes the envelope without `signature`, hashes those bytes, then inserts `signature` and serializes the complete envelope canonically). The receiver rejects noncanonical received bytes instead of accepting parse-then-canonicalize: canonical wire bytes make the signed input unique and avoid Go/JS parser disagreements. Before signature verification, validate strict UTF-8 (no replacement), reject duplicate object keys at every depth, and reject invalid JSON, noncanonical encoding, and invalid number forms. After removing the `signature` member, re-encode canonically and hash those bytes; never hash a permissively parsed object with silently collapsed keys. Canonical JSON is defined exactly: UTF-8; no insignificant whitespace; object keys sorted by Unicode code point; arrays in order; strings in shortest JSON-escaped form (escape `"` as `\"`, `\` as `\\`, and U+0000-U+001F as `\b`, `\t`, `\n`, `\f`, `\r` when applicable or lowercase `\u00xx` otherwise; do not escape `/`, other Unicode scalars, `<`, `>`, or `&`; reject unpaired surrogates; encode other scalars as UTF-8); numbers restricted to integers in decimal ASCII with no leading zeros (contract payloads carry no fractional or exponent-form numbers; a non-integer number makes the frame invalid, not re-serialized); no trailing newline. Empty body: SHA-256 of zero bytes = `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. |

- **Signature**: ed25519 (pure EdDSA over the base string, no prehash) by the device private key, transmitted base64url **without padding** (RFC 4648 §5).
- **Nonce**: 16 bytes from a CSPRNG, base64url without padding (22 chars).
- **Replay rules**: reject when `|server_now - timestamp| > 300` seconds; reject a nonce already seen for this device within the window. For each valid authenticated request/frame, admit nonce with an atomic insert-if-absent on `(device_id, nonce)` in the same transaction as the replay check before any effect; a concurrent duplicate loses, including across backend workers. Retain entries at least for the full 300-second acceptance window after their timestamp has aged out; prune only after they cannot pass the window. Before authentication, skewed, replayed, or invalid material receives the same generic pre-auth failure (§5.3); after authentication, an invalid frame is handled at the socket boundary per §5.3. Both sides are expected to run NTP-synced system clocks (macOS and standard server images do this by default). The 300-second window tolerates ordinary skew; a device whose clock is off by more than the window cannot connect or redeem until it resyncs - this is a deliberate fail-closed behavior, not an error to code around by widening the window locally.

### 3.2 Where signature and nonce appear

- **Redeem and connect (HTTP)**: four headers - `X-Waldo-Device-Id` (omit on redeem, where the device is not yet identified), `X-Waldo-Timestamp`, `X-Waldo-Nonce`, `X-Waldo-Signature`.
- **Socket frames**: device-to-backend envelope fields `timestamp`, `nonce`, `signature` (§4); backend-to-device frames carry none of these signature fields. Direction of trust: the device signs every frame it sends; the backend authenticates to the device by TLS server authentication - the device dials only the configured backend origin, verifies the certificate chain at connect, and aborts the socket on verification failure. Message-layer backend signing is not in v0.2.3; adding it is a version bump (§8).

### 3.3 Heartbeat, capabilities, offline behavior

- **Heartbeat**: device sends a `heartbeat` protocol message every 30 seconds on the live socket and immediately on reconnect, carrying `device_id`, `contract_version`, `declared_capabilities`, `outbox_depth` (count of outbox rows not yet `receipted`). Heartbeats are volatile: never spooled, never receipted, a lost heartbeat is repaired by the next one. `last_seen_at` drives the console device page and Waldo's honest "your Mac is offline" state.
- **Capability advertisement** on connect uses the exact query declaration in §3 and on heartbeat uses the same sorted JSON array as redeem (§2). Waldo must not send a command class the device has not declared.
- **Offline behavior**: while disconnected, the daemon spools outbound results in its durable outbox; the backend queues inbound commands with per-command TTLs and delivers on reconnect. Neither side fabricates progress for the other.

## 4. Message envelope and closed wire schemas

All socket frames are one canonical UTF-8 JSON object, max 8192 bytes including `signature`, one frame per WebSocket text message; binary, fragmented aggregate over the limit, JSON arrays at top level and unknown fields reject. No nested `payload` duplicates envelope fields. The five message types have an outer envelope and exactly one nested `payload` object as described here; §5's earlier `{ command_id, ... }` notation meant protocol data, not another copy of those envelope fields. A field not explicitly permitted for that type is forbidden. `null` is not a substitute for absent. Unsigned backend frames omit all three signing fields; signed device frames require all three.

| Outer field | Scope, type and limit |
|---|---|
| `contract_version` | all; literal string `0.2.3`; mismatch yields `version_mismatch` after authentication |
| `type` | all; exactly one of `command`, `ack`, `result`, `receipt`, `heartbeat` in the permitted direction |
| `message_id` | all; 26-char uppercase Crockford-base32 ULID; identity of THIS frame, not a referenced frame; backend chooses for command/receipt, device for ack/result/heartbeat |
| `device_id`, `owner_id` | all; nonempty ASCII identifiers, 1-128 bytes each, `[A-Za-z0-9_-]` only; must match authenticated binding |
| `command_id` | command/ack/result/receipt only; same identifier grammar and 1-128 byte bound; references the command; absent for heartbeat |
| `revision` | command/ack/result/receipt only; integer exactly `1`; absent for heartbeat; a corrected command has a new `command_id` |
| `idempotency_key` | command/ack/result/receipt only; same identifier grammar and 1-128 byte bound, copied unchanged from command |
| `class` | command only; `machine_state_query` or `notify_local` |
| `expires_at` | command only; Unix seconds integer > 0; reject at or after expiry; backend limits TTL per §6 |
| `payload` | all; required JSON object, exact shape per §5.1 and §5.2 |
| `timestamp` | device-to-backend only; Unix seconds integer > 0; its decimal ASCII representation is the signing field |
| `nonce` | device-to-backend only; base64url without padding of exactly 16 random bytes, 22 ASCII characters |
| `signature` | device-to-backend only; base64url without padding of exactly 64 ed25519 signature bytes, 86 ASCII characters |

A sender generates a fresh `message_id` for each new logical protocol frame. A retransmission of that logical frame preserves all fields except fresh `timestamp`, `nonce`, and `signature`. For device-origin volatile `ack`/`heartbeat`, a new send is a new message, not a retransmission. A backend receipt has its OWN outer `message_id`; its `payload.result_message_id` points to the result it confirms. This resolves the former receipt `message_id` collision. Both sides validate that receipt `command_id`, `revision`, `idempotency_key`, `device_id` and `owner_id` match the result row named by `result_message_id`. The logical fingerprint (§6) includes the nested payload and all stable outer fields. All strings are preserved as sent, never silently normalized (NFC and decomposed forms remain distinct). Limits count UTF-8 bytes before JSON escaping, except the frame-byte limit above. Any ID not otherwise bounded below uses the identifier grammar above. Arrays have no duplicate entries.

## 5. Vocabulary (closed taxonomy)

### 5.1 Protocol message types and exact payloads

| Type | Direction | Nested `payload` object (only these keys) | Durability |
|---|---|---|---|
| `command` | backend -> device | class-specific keys in §5.2 | Durable: journal before execution |
| `ack` | device -> backend | `state` required enum `accepted`, `rejected`, `expired`; `reason` required only for `rejected` or `expired`, one of §5.3 codes; absent for `accepted` | Volatile, best-effort, never spooled; backend redelivers if lost |
| `result` | device -> backend | `status` required enum `answered`, `delivered`, `failed`; `answer` required only for `answered` and follows §5.2; `reason` required only for `failed` and is a §5.3 code, including `processing_failed` for query-processing failure; `answer` absent for other statuses | Single durable outbox result per `(command_id, revision)`; cleared by receipt |
| `receipt` | backend -> device | `result_message_id` required ULID, names result frame; `received_at` required positive Unix-seconds integer | Confirms matching outbox row; duplicate is no-op; unknown result closes socket without typed wire response, local diagnostic `unknown_message` (§5.3) |
| `heartbeat` | device -> backend | `declared_capabilities` required ordered nonempty subset of §5.2 classes, 1-2 entries; `outbox_depth` required integer 0..1000000 | Volatile; never spooled or receipted |

`ack`, `result`, and `receipt` are not revisions of a command and never carry `class`. A command yields at most one durable outbound message, its result, as built on the Kennel side. `heartbeat` carries device identity and contract version only in the outer envelope. `state`, `status`, `reason`, `answer`, `declared_capabilities`, `outbox_depth`, `result_message_id`, and `received_at` are nested in `payload` ONLY, never copied outside it.

Examples below are canonical JSON objects for each type. Command and receipt are complete backend frames. Ack, result and heartbeat omit `timestamp`, `nonce` and `signature` to show payload placement only and MUST NOT be sent as shown; Appendix A has complete signed-input frames and their computed signatures.

```json
{"class":"machine_state_query","command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","expires_at":1790700000,"idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","owner_id":"owner_1","payload":{"query_id":"query_1","query_kind":"session_status"},"revision":1,"type":"command"}
{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAW","owner_id":"owner_1","payload":{"state":"accepted"},"revision":1,"type":"ack"}
{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX","owner_id":"owner_1","payload":{"answer":{"query_id":"query_1","query_kind":"session_status","state":"idle"},"status":"answered"},"revision":1,"type":"result"}
{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAY","owner_id":"owner_1","payload":{"received_at":1790200802,"result_message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX"},"revision":1,"type":"receipt"}
{"contract_version":"0.2.3","device_id":"dev_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAZ","owner_id":"owner_1","payload":{"declared_capabilities":["machine_state_query","notify_local"],"outbox_depth":0},"type":"heartbeat"}
```

### 5.2 Frozen command classes (K0) and answers

| Class | Nested command payload, all keys required | Effect |
|---|---|---|
| `machine_state_query` | `query_id`: ID 1-128 bytes; `query_kind`: `session_status`, `attempt_status`, `worktree_watch` | Read only durable daemon state; unknown kinds reject, no inferred answers |
| `notify_local` | `notification_id`: ID 1-128 bytes; `title`: string 1-120 UTF-8 bytes; `body`: string 1-1024 UTF-8 bytes; `severity`: `info`, `warning`, or `error` | Display text on Mac (island or native), execute nothing |

For a successful `machine_state_query`, `result.payload` has exactly `status` with value `"answered"` and `answer` with exactly `query_id`, `query_kind`, and `state`. A real processing failure instead yields `result.payload` with exactly `status` set to `"failed"` and `reason` set to `"processing_failed"`, with no `answer`; the failure never claims a machine-state answer. A `delivered` result is invalid for this class. `query_id` and `query_kind` echo the command. `state` is one of `idle`, `running`, `done`, `failed`, `unknown` for `session_status` and `attempt_status`; for `worktree_watch`, `state` is `watching`, `stopped`, or `unknown`. These are machine-state labels, never invented progress. If the daemon lacks durable evidence for a label, return an `answered` result with `state:"unknown"` rather than infer. An inability to perform the query itself is a processing failure, not a missing state. For `notify_local`, successful `result.payload.status` is `delivered`, with no `answer`; uncertain delivery after interruption is `failed` with reason `delivery_unknown`. A result type/status inconsistent with the command class rejects as `invalid_shape`. No other class exists in v0.2.3; in particular no approval, accept, file, shell, or task class. Adding a class or answer field is a version bump (§8).

### 5.3 Error taxonomy and authentication boundary

Before successful authentication (including malformed UTF-8/JSON, invalid key/signature, wrong code, expired or racing code, throttling, nonce replay or skew, unknown device, and failed connect declaration), HTTP surfaces always return the same status `401`, content type `application/json`, body bytes `{"error":"invalid_request"}` (UTF-8, no newline), with no distinguishing header or redirect; socket handshake rejects with the same status/body before upgrade. TLS handshake failure produces no application response. No typed reason leaves the backend before authentication.

After an authenticated socket is established, a **device-origin `ack`** is sent only in response to a backend-origin `command` whose outer envelope has valid bound `device_id`, `owner_id`, `command_id`, `revision=1`, and `idempotency_key`, so the ack can unambiguously name it. Rejected command shape/class/expiry or version is `ack.payload` with `state:"rejected"` and reason `invalid_shape`, `idempotency_conflict`, `unknown_command`, or `version_mismatch` as applicable; expired commands use `state:"expired",reason:"expired"`. No effect occurs for a rejection. A valid accepted command gets `state:"accepted"`. A query processing failure after acceptance sends a `result` with `status:"failed",reason:"processing_failed"`; `notify_local` uncertain delivery sends `status:"failed",reason:"delivery_unknown"`. Neither `processing_failed` nor `delivery_unknown` is an ack reason.

An authenticated frame with absent, invalid or mismatched binding/command reference that prevents a trustworthy ack, an invalid **device-origin** `ack`/`result`/`heartbeat`, or a **backend-origin** `receipt` naming an unknown result (`unknown_message`) causes terminal socket close with no typed wire response or effect. The receiving side may log a typed server/local diagnostic (`invalid_shape`, `unknown_message`, `idempotency_conflict`, or `version_mismatch`) but does not send an `ack` in the wrong direction. A well-formed duplicate receipt matching a stored receipt tombstone is a no-op (§6). A post-auth device-origin frame with a reused message ID and mismatched fingerprint likewise closes without receipt or effect. No standalone `error` frame type is added. All errors after authentication are either the specified typed command ack/result or a terminal close with diagnostic, never an unrouteable typed ack.

## 6. Ordering, replay, reconciliation

- Per `command_id` there is exactly one revision; within it, the protocol order is `command -> ack -> result -> receipt`. Across commands, no ordering is assumed.
- On reconnect, the backend drops expired commands from its undelivered queue and never sends them. The device drains its outbox (oldest `seq` first, deduplicated by `idempotency_key` and `message_id`), then reconciles accepted but unresulted commands, then accepts new commands. On receipt or redelivery of an expired command, the device sends a volatile post-auth `expired` ack and neither executes it nor extends its TTL. Nothing new issues before in-flight work reconciles.
- A redelivered command (same `command_id`, `revision`, `idempotency_key`, same payload fingerprint) maps to the journaled outcome: if a `result` exists, the device re-sends that same result message - identical `message_id`, `command_id`, `revision`, `idempotency_key`, and payload, freshly signed with a new `timestamp` and `nonce` (dedup is by `message_id`, never by signature bytes, so a re-send has fresh signing fields); if not, it re-acks and continues. One effect, ever.
- An interrupted `notify_local` whose delivery state is unknown is reported `failed` with reason `delivery_unknown` on reconnect, never silently assumed delivered.
- The backend must not retry a command into a second effect: retry = redeliver with the same `idempotency_key`. On each receiving side, a stored dedup row is keyed by `(device_id, message_id)` and carries a SHA-256 fingerprint of the canonical logical frame excluding only `timestamp`, `nonce`, and `signature`; an identical re-send is a no-op or maps to the journaled result, while a reused key with a different fingerprint is `idempotency_conflict` after authentication. The command journal also keys by `(device_id, command_id, revision)` and maps the stable device-scoped `idempotency_key` and command payload fingerprint to the one outcome; redelivery under a new message ID still cannot re-execute it. The device journals command acceptance before any effect, and persists a result and its outbox row in one transaction. The backend persists the result and its device-scoped message-ID fingerprint/dedup row in one transaction before issuing a receipt. On a duplicate result with the same `(device_id,message_id)` and identical logical fingerprint, including after restart, the backend does not reapply it and issues a fresh backend-origin `receipt` with a new receipt `message_id`, referencing the same result `message_id`; a mismatched fingerprint closes with no receipt. The receipt is sent only after the backend transaction commits, and may be re-issued any number of times until the device confirms by clearing the outbox. The backend durably retains, per paired device, the accepted result's `(device_id,message_id)` logical fingerprint and the minimal result/command metadata needed to re-issue `receipt` for as long as that device may authenticate and retransmit its outbox, including across restart and arbitrarily long disconnection after command expiry. Receipt issuance, socket delivery and aggregate heartbeat depth are not proof of receipt application. Every matching duplicate gets another receipt; conflicting fingerprints never do. A live device's dedup record is never evicted for age or size; apply backpressure to new work instead. After device revocation or an explicitly versioned decommission that prevents all further authentication/replay for that identity, retain through the 300-second replay horizon, then prune. Do not retain full answer bytes solely for this dedup purpose when the fingerprint and receipt metadata suffice. The device applies a receipt, removes the matching outbox row and stores a receipt tombstone in one transaction; a duplicate matching the tombstone is a no-op. Tombstones are bounded to 100000 entries per device and retained until at least `expires_at + 300` seconds for that command and until no matching in-flight result remains; never merely 300 seconds after receipt; commands have a maximum TTL of 86400 seconds from issue, and greater TTLs reject as `invalid_shape`. If the cap is reached, reject new commands as `invalid_shape` rather than evict live tombstones. Once tombstones age out, the separately retained command journal still prevents re-execution of the same `command_id` and `idempotency_key`. On the Kennel-side store, journal and outbox identity/uniqueness are scoped by `device_id`: the journal key is `(device_id, command_id, revision)`, with `(device_id, idempotency_key)` unique; the outbox has `(device_id, message_id)` and `(device_id, command_id, revision)` unique, while preserving its durable `seq` order and one result row per command/revision. The already-shipped 0162 migration is never rewritten in place. A new forward migration rebuilds the existing bridge tables and indexes, copies every existing row with a valid associated device identity, validates counts and constraints, and commits atomically; any row whose device cannot be determined or any uniqueness conflict rolls back the entire migration for explicit repair, not silent row loss or an arbitrary device assignment. No parallel active bridge tables or dual-write path are introduced. The migration number is chosen only after checking the live ledger and branches, including K4 work. This is an implementation requirement as built on the Kennel side, not a change to the backend wire schema. The result remains one durable outbox row per `(device_id, command_id, revision)`.

## 7. Authority and privacy invariants

- Pairing authenticates transport only. No message in this contract grants, widens, or exercises owner authority. `notify_local` is display-only, bounded text sent to the paired device; it does not request execution or owner-scoped approval. Ashish chose the Option A issuance policy on 2026-09-29 at 11:34:36 PM IST; this contract defines a permitted policy shape, not a standing grant to send any actual notification. Pairing is not a grant to display arbitrary content. For `notify_local`, the backend may issue only when an authenticated, separately sourced owner request explicitly asks for a local Mac notification of the given content, or a separately recorded and still-valid owner grant covers the exact notification class, source and device. A peer message, model output, pairing, or general channel access alone supplies no such permission. Before issuance, the backend verifies owner-device binding and the permitted content scope; default content is a short, neutral, nonsensitive status line, with no secrets, transcript excerpts, personal health/financial details or third-party private content. The notification is plain display text, with URLs not made actionable, no execution or owner approval. Enforce the §5.2 title/body bounds and a per-device issue cap of 5 notifications per 60 seconds and 20 per hour, counting retries by `notification_id` only once; reject overflow without queueing for later display. If a grant exists, the owner can revoke it; disabled devices receive none. This is issuance/display policy, not a new authority-bearing command class or permission inferred from the wire.
- Material transitions stay native on the Mac under the transitional rule (ADR 0018 §4). A channel message is not owner proof.
- Query answers contain only the machine state the query asked for. No personal memory, health data, transcript content, or unrelated local state crosses the bridge in v0.2.3.
- Long-lived secrets never appear in any message, log line, or stored payload. Signatures, nonces, and public keys are not secrets; private keys and pairing codes are.

## 8. Versioning

- The contract version is negotiated at redeem and re-declared on every connect (§3 query declaration). On redeem and connect, the device declares `contract_version` as `0.2.3`; `accepted_contract_version` on redeem is `0.2.3`. Before authentication, a mismatch or unknown version receives the generic failure (§5.3); after authentication, it receives `version_mismatch`, never a guessed fallback.
- New classes, payload fields, or changed semantics require a new contract version accepted by both sides. Additive optional fields are still a version bump: closed shapes reject unknown fields by design. (v0.2's envelope additions over v0.1 were exactly such a bump.)

## 9. K0 exit criteria (both lanes)

1. Pair: code mint -> redeem -> signed connect -> heartbeat; console shows the device; revoke kills it (device observes unpaired).
2. Round trip: from a Waldo channel, a `machine_state_query` is answered from real daemon state and a `notify_local` surfaces on the Mac, each acked, resulted, and receipted.
3. Failure matrix, scripted and logged with exact commands and durations: duplicate delivery (one effect), device offline beyond `expires_at + 300`, backend restart, same result retransmitted, receipt re-issued and device outbox cleared; lost receipt and backend restart with re-receipt, disconnect/reconnect mid-round-trip, offline queue drain without duplicates, parallel code redemption, redeem expiry and rate limits, revoked device rejected, forged/stale/widened frames rejected before any local action, notification issuance outside an owner grant rejected.

## Appendix B. Kennel migration evidence at v0.2.3 drafting (2026-09-29 11:31 PM IST)

Live `outcome-loop` tip from `git ls-remote`: `0f866cd054f68c17f9d2410af4b8cdd26a157d84`. The [0162 bridge migration](https://github.com/waldoco/Waldo-Kennel/blob/0f866cd054f68c17f9d2410af4b8cdd26a157d84/backend/internal/storage/sqlite/migrations/0162_device_bridge.sql) uses globally unique message/command keys and has no `device_id` on its outbox/journal. The [0163 capture migration](https://github.com/waldoco/Waldo-Kennel/blob/0f866cd054f68c17f9d2410af4b8cdd26a157d84/backend/internal/storage/sqlite/migrations/0163_capture_source_plane.sql) is present. The [migration ledger test](https://github.com/waldoco/Waldo-Kennel/blob/0f866cd054f68c17f9d2410af4b8cdd26a157d84/backend/internal/storage/sqlite/migrate_burned_versions_test.go) lists `162: 0162_device_bridge.sql` and `163: 0163_capture_source_plane.sql`, with no 164 entry. The live GitHub recursive tree for that exact tip lists migration files through 0163 and no 0164 file. Thus 0163 is the highest used number in `outcome-loop` and **0164 is the next free number on that branch at this readback**, not a reservation across other active branches or K4 work. Before implementation the Mac agent must recheck all branches and the ledger, allocate the then-unclaimed number, and show a collision/readback migration test. Existing `TestMigrationVersionLedger` checks filename/number uniqueness and ledger presence, not the new device-scoped row-copy migration; that new test has not run and must not be claimed green. No v0.2.3 migration has been committed.

## Appendix A. Golden signature vectors (normative bytes)

The fixed test-only ed25519 private seed is 32 bytes `000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f`; never use this key outside tests. All base strings shown with `\n` denoting one LF byte (0x0A), not two backslash characters. The listed public key, canonical JSON, lowercase SHA-256 digest and expected 64-byte signature (base64url without padding and hex) are generated from this seed. The redeem vectors use a fixed syntactically valid test code; the bytes are public and must never be accepted as a live code. The frames are test vectors, not live traffic: their timestamps will be stale. Receivers enforce the actual current 300-second window in live use. JSON vectors exclude the `signature` field as required by §3.1; inserting the listed base64url signature at top level and canonically serializing creates the on-wire frame. Tests must verify signature, not merely compare digests. `signature` is absent from HTTP bodies.


Public key, base64url: `A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg`

Nonce for each independent test: `AAECAwQFBgcICQoLDA0ODw` (test fixtures reuse a nonce across independent verification tests; replay-admission tests must not).

### 0. Redeem HTTP signing

Canonical JSON (signature excluded):
```json
{"code":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8","contract_version":"0.2.3","declared_capabilities":["machine_state_query","notify_local"],"device_pubkey":"A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg","label":"Test Mac"}
```
`body_sha256`: `90d6daa8d89edfae3a13f3584eb36cf04155539bf5c2c2ccf2f73de122ba6175`

Base string (each shown `\n` denotes LF):
```text
1790200800\nAAECAwQFBgcICQoLDA0ODw\nPOST\n/devices/redeem\n90d6daa8d89edfae3a13f3584eb36cf04155539bf5c2c2ccf2f73de122ba6175
```
Expected signature, base64url without padding: `VlVfMNC0TTbHKXsDsjGjWyBHnaVAMSud0b_DlEukL78UzWBdJhs_knA09uH0NSzsuFbWpLSU-xYhqKEsFGPbDw`

Expected signature bytes, lowercase hex: `56555f30d0b44d36c7297b03b231a35b20479da540312b9dd1bfc3944ba42fbf14cd605d261b3f927034f6e1f4352cecb856d6a4b494fb1621a8a12c1463db0f`

### 1. Accepted ack

Canonical JSON (signature excluded):
```json
{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAW","nonce":"AAECAwQFBgcICQoLDA0ODw","owner_id":"owner_1","payload":{"state":"accepted"},"revision":1,"timestamp":1790200800,"type":"ack"}
```
`body_sha256`: `7650d9debd01414a3cc82e11e117a8faad12d59de977d37010c49bf91d66eb4f`

Base string (each shown `\n` denotes LF):
```text
1790200800\nack\n01ARZ3NDEKTSV4RRFFQ69G5FAW\nAAECAwQFBgcICQoLDA0ODw\n7650d9debd01414a3cc82e11e117a8faad12d59de977d37010c49bf91d66eb4f
```
Expected signature, base64url without padding: `c2GwY1y5gWNCSPdf9Ufb8IIRgMGx_t9wpOi02pbKlHcC-QdOb7i7R5wp-cwmk-u2Lb-mO6ah2hugIFYflTQNBw`

Expected signature bytes, lowercase hex: `7361b0635cb981634248f75ff547dbf0821180c1b1fedf70a4e8b4da96ca947702f9074e6fb8bb479c29f9cc2693ebb62dbfa63ba6a1da1ba020561f95340d07`

### 2. Answered result

Canonical JSON (signature excluded):
```json
{"command_id":"cmd_1","contract_version":"0.2.3","device_id":"dev_1","idempotency_key":"idem_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAX","nonce":"AAECAwQFBgcICQoLDA0ODw","owner_id":"owner_1","payload":{"answer":{"query_id":"query_1","query_kind":"session_status","state":"idle"},"status":"answered"},"revision":1,"timestamp":1790200800,"type":"result"}
```
`body_sha256`: `1a0b0591d76e18bcf22772e2b9bdb68d63da0e55dc671ced8397941ec657210c`

Base string (each shown `\n` denotes LF):
```text
1790200800\nresult\n01ARZ3NDEKTSV4RRFFQ69G5FAX\nAAECAwQFBgcICQoLDA0ODw\n1a0b0591d76e18bcf22772e2b9bdb68d63da0e55dc671ced8397941ec657210c
```
Expected signature, base64url without padding: `Ur-XO9PNQy3Kszxe7A_SJp7GssLVSKp6wkZUqEZAYHkKGon0IqbLhA-vhD1ya_o-HQQkb2wOB9Sgyyd8PWpDAA`

Expected signature bytes, lowercase hex: `52bf973bd3cd432dcab33c5eec0fd2269ec6b2c2d548aa7ac24654a8464060790a1a89f422a6cb840faf843d726bfa3e1d04246f6c0e07d4a0cb277c3d6a4300`

### 3. Heartbeat

Canonical JSON (signature excluded):
```json
{"contract_version":"0.2.3","device_id":"dev_1","message_id":"01ARZ3NDEKTSV4RRFFQ69G5FAZ","nonce":"AAECAwQFBgcICQoLDA0ODw","owner_id":"owner_1","payload":{"declared_capabilities":["machine_state_query","notify_local"],"outbox_depth":0},"timestamp":1790200800,"type":"heartbeat"}
```
`body_sha256`: `431e3fa8304bcdca049b84a0545cd82a6b7ed1f9596f18e227adc8b9b42c931f`

Base string (each shown `\n` denotes LF):
```text
1790200800\nheartbeat\n01ARZ3NDEKTSV4RRFFQ69G5FAZ\nAAECAwQFBgcICQoLDA0ODw\n431e3fa8304bcdca049b84a0545cd82a6b7ed1f9596f18e227adc8b9b42c931f
```
Expected signature, base64url without padding: `1DzrSOApCr5iU_sTuJFozlndCY-A1EhY0XVtYGf07nP_OoW_Q8bI1a9OakAvlfJBBfBQb1NDevyyH4Q3tbWzDA`

Expected signature bytes, lowercase hex: `d43ceb48e0290abe6253fb13b89168ce59dd098f80d44858d1756d6067f4ee73ff3a85bf43c6c8d5af4e6a402f95f24105f0506f53437afcb21f8437b5b5b30c`

### 4. Redeem HTTP Unicode/HTML-escape fixture

This signed body tests direct UTF-8 for `é`, `𐀀`, `<`, and `>` instead of Go's default HTML escaping or JavaScript's UTF-16 code-unit sorting. The same key/nonce applies as above.

Canonical JSON (signature excluded):
```json
{"code":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8","contract_version":"0.2.3","declared_capabilities":["machine_state_query"],"device_pubkey":"A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg","label":"Café <Go> 𐀀"}
```
`body_sha256`: `cc34011527d0ae10bad46dcc1977f79898a14b2b4b71ce25f3d7b1280bb6e00a`

Base string (each shown `\n` denotes LF):
```text
1790200800\nAAECAwQFBgcICQoLDA0ODw\nPOST\n/devices/redeem\ncc34011527d0ae10bad46dcc1977f79898a14b2b4b71ce25f3d7b1280bb6e00a
```
Expected signature, base64url without padding: `LN1dFNztIWTdRKIPPqWtUolXKHhmGygGapStEuvsxIftaLrwwGfo9V_cED3D5T1fQufFL4cAYsKjoBTBe4PhAA`

Expected signature bytes, lowercase hex: `2cdd5d14dced2164dd44a20f3ea5ad5289572878661b28066a94ad12ebecc487ed68baf0c067e8f55fdc103dc3e53d5f42e7c52f870062c2a3a014c17b83e100`

### 5. Serializer-only cross-language ordering and escaping fixture (not a protocol frame)

The following is a canonicalizer conformance fixture, not a valid message schema. It tests code-point ordering (U+E000 sorts before U+10000 whereas UTF-16 code-unit order can disagree), distinct composed/decomposed keys (no normalization), direct UTF-8, and Go HTML-escaping differences.

```json
{"é":"combining","text":"<>&/\n\t\"\\","é":"café","":"private","𐀀":"plane"}
```

Expected UTF-8 bytes, lowercase hex: `7b2265cc81223a22636f6d62696e696e67222c2274657874223a223c3e262f5c6e5c745c225c5c222c22c3a9223a22636166c3a9222c22ee8080223a2270726976617465222c22f0908080223a22706c616e65227d`

