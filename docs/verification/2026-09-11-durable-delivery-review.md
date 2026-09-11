# Independent review — durable delivery

- **Date:** 11 September 2026
- **Scope requested:** artifact/acceptance binding, idempotency, interruption, reconciliation, destination safety
- **Base:** PR #103 head `e573a7229`, reviewed on `codex/pr103-launch-repairs`
- **Sources reviewed:** `backend/internal/service/outcome/delivery.go`,
  `backend/internal/artifactstore/export.go`,
  `backend/internal/storage/sqlite/store/outcome_delivery_store.go`,
  `backend/internal/storage/sqlite/queries/outcome_delivery.sql`,
  `backend/internal/domain/outcome_delivery.go`, and the delivery call site in
  `backend/internal/daemon/daemon.go`.

This is a source and automated-boundary review. **No live delivery to a real
owner destination was performed, and no owner accepted any result.** Nothing in
the reviewed area was changed except test fidelity; the findings below are
reported, not repaired, because two of the three require a product decision
this review is not authorized to make.

## Verdict

The delivery write path is sound. The transfer is atomic, the bytes are
verified against the retained manifest before they are published, and an
accepted delivery cannot be produced without the owner's matching
AcceptanceDecision — checked once in the service and again, independently,
inside the artifact store.

The weakness is not in transferring; it is in **what the ledger can honestly say
afterwards**. Delivery records its result after the filesystem effect commits,
and nothing ever reads the destination back. Every gap below is a variant of
that single seam.

## What holds

### Artifact and acceptance binding — sound

An accepted delivery is refused unless all of the following hold. The service
checks them, and `artifactstore.bindAcceptance` re-checks the same facts from
the receipt rather than trusting the caller:

- the receipt belongs to this Outcome, Attempt and WorkUnit lineage
  (`delivery.go:132`);
- the requested artifact version *is* the retained version — delivery never
  means "the newest result" (`delivery.go:135`);
- retention is complete (`delivery.go:138`);
- the decision is `accept`, by `user`, for this Outcome and for the *same
  Contract revision the artifact was produced under* (`export.go:88-101`);
- the decision names this exact artifact version (`export.go:105`);
- the named acceptance is still the Outcome's latest decision, so a later
  rework or reopen withdraws delivery authority (`delivery.go:283`).

A draft delivery may carry no AcceptanceDecision at all, and the exported
manifest records `"disposition": "draft"`, so an unaccepted bundle cannot be
mistaken for an accepted one on disk. Export is never merge, publication or
acceptance.

Existing coverage: `TestRequestDeliveryBindsCurrentAcceptanceAndIsIdempotent`
proves the accepted path and the post-rework refusal.

### Idempotency — sound

Three layers, and they agree:

1. the service resolves a repeated request key before doing anything, returning
   the durable row rather than transferring again (`delivery.go:104`);
2. `CreateOutcomeDelivery` resolves the same key inside its transaction, and
   converts a lost cross-process unique-key race into the same replay semantics
   instead of leaking a constraint error (`outcome_delivery_store.go:51`);
3. after creating, the service **re-reads the durable row and returns early if
   another writer owns it**, so only the request that owns the pending row
   touches the filesystem (`delivery.go:175`).

Reusing a key for different semantics is a durable conflict, including while
the first request is still pending. `CompleteOutcomeDelivery` is conditional on
`state = 'pending'`, so a delivery terminalizes exactly once.

Two concurrent requests with *different* keys and the same destination are
arbitrated by the filesystem: one wins the rename, the other fails with
`DELIVERY_DESTINATION_CONFLICT`. No double transfer is possible.

### Interruption of the transfer itself — sound

`Export` stages into a sibling directory, writes each file, reads it back and
compares against the retained digest, writes the manifest, then commits with a
single `os.Rename`. A failure at any point removes the staging directory. The
destination is therefore only ever whole or absent — an owner can never find a
half-written bundle and mistake it for a result.

New coverage:
`TestRequestDelivery_ACancelledRequestLeavesNothingAtTheDestination` proves a
cancelled request leaves neither a destination nor staging debris, and that the
owner can simply retry.

## Findings

### DLV-01 · P1 · A committed transfer that misses its ledger write is recorded as failed forever

**Where:** `delivery.go:213` (completion follows the commit),
`delivery.go:220` `ReconcileDeliveries`, `outcome_delivery.sql:26`.

The rename that publishes the bundle and the row that records it are two steps.
A daemon that stops between them leaves the bytes delivered and the row pending.
On restart, `FailPendingOutcomeDeliveries` closes every pending row as
`failed` / `DELIVERY_INTERRUPTED`.

Failing an unproven transfer is the right default. The problem is that the
proof is *right there and unread*: the destination contains
`KENNEL-EXPORT.json`, naming the exact Attempt, artifact version, Contract
revision and AcceptanceDecision. Reconciliation never opens it.

The owner is then stuck. The ledger says the delivery failed; retrying to the
same destination returns `DELIVERY_DESTINATION_CONFLICT`, which reads as an
unrelated problem; and the only way to learn the truth is to inspect the
filesystem by hand. Delivery history — the thing the ledger exists to be — is
wrong, and stays wrong.

**Proved by:** `TestReconcileDeliveries_CannotTellACommittedTransferFromAnAbandonedOne`.

**Recommended:** have reconciliation read the destination manifest for each
pending row and compare it against that row's Attempt, artifact version and
AcceptanceDecision. On an exact match, close the row as succeeded, recording
that the result was established by reconciliation rather than observed. On any
mismatch or absence, close it as interrupted exactly as now. This claims no
success Kennel cannot evidence — it reads Kennel's own manifest — but it is a
product decision about what "succeeded" may mean, so it is left to the owner.

A narrower alternative, if reconciliation must stay purely conservative: keep
failing the row, but record the observed manifest digest on it so the Mission
can tell the owner "an artifact matching this request is already at the
destination" instead of an opaque conflict.

### DLV-02 · P2 · Destination safety checks the last path component only

**Where:** `delivery.go:306` and `export.go:143`.

Both the service and the export refuse a destination that *is* a symlink. No
component above it is examined, so a symlinked parent silently redirects the
whole transfer: `os.MkdirAll` follows it and the bundle lands somewhere other
than the path the owner named and the path the ledger records.

**Proved by:** `TestRequestDelivery_ASymlinkedParentIsNotRefusedTheWayALeafSymlinkIs` —
a symlink as the destination is refused, and the same link one level up is
followed to a successful delivery.

The trust model matters here: the owner types the destination, so this is not
an escalation so much as a truthfulness gap — `delivery.destination` and the
`manifestPath` in the receipt name a path that is not where the bytes are.

**Recommended:** do not blanket-refuse symlinked parents. Common install layouts
legitimately contain them (`/tmp` is a symlink on macOS), so refusing would
break ordinary destinations. Resolve the parent with `filepath.EvalSymlinks`
and persist the *resolved* destination on the delivery row, so the receipt tells
the owner where the artifact actually is. Refuse only when the parent cannot be
resolved.

### DLV-03 · P3 · `FailPendingOutcomeDeliveries` is unscoped and safe only by call-site ordering

**Where:** `outcome_delivery.sql:26` (`WHERE state = 'pending'`, no other
predicate), called from `daemon.go:453`.

The query closes every pending row in the database, with no scope to the
daemon, process or start time. Today that is safe purely because the single
call site runs during daemon construction, before the listener serves, so no
delivery of this daemon's can be in flight.

Nothing in the query or the store enforces that. A second daemon instance
against the same `~/.kennel` database, or any future caller that reconciles on
a timer rather than at startup, would mark a live in-flight delivery failed
while its export is still running — and the owning request would then get
`delivery … became stale before terminalization` (`delivery.go:245`), a 500,
even though the transfer succeeded.

**Recommended:** scope the predicate to rows requested before this daemon's
start, or mark pending rows with the owning daemon's run identity. No behaviour
change today; it removes a trap for the next caller.

## Test fidelity repaired during this review

`memoryDeliveryStore.FailPendingOutcomeDeliveries` was a stub returning `0`, so
no service-level test could exercise reconciliation at all. It now mirrors the
SQLite query, including its unscoped `pending` predicate. This is what let
DLV-01 be proved rather than asserted.

## Evidence and limits

Automated, on this branch: `go build ./...`, `go vet ./...`, `go test ./...`,
and `go test -race` over `internal/service/outcome`, `internal/artifactstore`
and `internal/storage/sqlite/store` — all exit 0.

Not established here, and not claimed:

- no delivery to a real owner-chosen destination outside a temp directory;
- no packaged or native journey, and no daemon actually restarted — DLV-01's
  restart is modelled by rewinding the ledger row to `pending`, which is the
  state a crash leaves, not a crash;
- no owner reviewed or accepted any delivered artifact;
- no cross-filesystem destination (the `os.Rename` commit is same-filesystem by
  construction; a destination on another volume is untested);
- no concurrent multi-daemon run against one database, so DLV-03 is reasoned
  from the query and its call site rather than reproduced.
