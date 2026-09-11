# Delivery recovery contract — 11 September 2026

Frontend-facing status and action contract for durable delivery, after DLV-01.
Generated sources are authoritative: `backend/internal/httpd/apispec/openapi.yaml`
and `frontend/src/api/schema.ts`.

Endpoints: `GET|POST /api/v1/outcomes/{outcomeId}/deliveries`,
`GET /api/v1/outcomes/{outcomeId}/deliveries/{deliveryId}`.

## What changed

One additive response field. No field was removed, renamed or retyped, and no
request shape changed.

### `delivery.completionSource` (new, string, omitted while pending)

| Value | Meaning |
|---|---|
| `observed` | The daemon watched the transfer resolve and then wrote the row. |
| `recovered` | The row was left pending by a crash between the filesystem commit and the ledger write. The result was established afterwards by reading the destination's manifest and re-digesting every delivered artifact against the retained result. |

A `recovered` success proves the bytes arrived. **Nobody watched them arrive**,
and the two are not the same evidence — which is why this is durable rather
than derived. Render it; do not hide it behind the state.

Absent means the delivery is still `pending`.

## Rendering the four states

`state` is unchanged (`pending | succeeded | failed | cancelled`). What is new
is that a `pending` row is now resolved on the next daemon start rather than
blanket-failed, so these are the combinations a client will see:

| `state` | `completionSource` | `failureCode` | What to show |
|---|---|---|---|
| `pending` | absent | absent | Transfer in flight, or awaiting the next daemon start. Offer nothing; do not offer a retry. |
| `succeeded` | `observed` | absent | Delivered. `manifestPath`, `fileCount`, `byteCount` describe it. |
| `succeeded` | `recovered` | absent | Delivered, confirmed by inspection after an interruption. Same affordances; say it was confirmed rather than watched. |
| `failed` | `recovered` | see below | Not delivered, or not provably delivered. The code says which. |
| `cancelled` | `observed` | `DELIVERY_CANCELLED` | The request was cancelled; nothing was left at the destination. A clean retry is safe. |

## Failure codes, and the owner's next move

Recovery never writes, never re-transfers and never removes anything, so for
every failure below the destination is exactly as reconciliation found it. The
action differs per code and the renderer must not collapse them:

| `failureCode` | Meaning | Owner's next move |
|---|---|---|
| `DELIVERY_INTERRUPTED` | The daemon stopped before the transfer finished and **nothing** is at the destination. | Request delivery again to the same destination. Safe: nothing is in the way. |
| `DELIVERY_RECOVERY_MISMATCH` | Something **is** at the destination and it is not this delivery — a missing or altered artifact, unrelated content with no manifest, or a manifest naming a different Attempt, artifact version, disposition, Contract revision or AcceptanceDecision. | Do **not** offer a same-destination retry as the primary action: it will be refused with `DELIVERY_DESTINATION_CONFLICT`, and forcing it would overwrite whatever is there. Show `failureDetail`, and offer a different destination. |
| `DELIVERY_RECOVERY_UNREADABLE` | The destination could not be read well enough to decide — an unparseable manifest, a permission error. | Same as mismatch. Not deciding is not the same as deciding against, and the wording should not imply the transfer failed. |
| `DELIVERY_RECOVERY_UNVERIFIABLE` | The retained result this delivery named is no longer available, so the destination has nothing authoritative to be checked against. | Nothing to recover. The artifact would have to be produced again. |
| `DELIVERY_DESTINATION_CONFLICT` | A **live** request found the destination already occupied. | Choose another destination. |

`failureDetail` is populated for every failure and is the only place the
specific reason appears. A failure card without it is not actionable.

## What a recovered success does not mean

Delivery is transfer. It is never merge, publication, or Outcome acceptance, and
recovery does not change that:

- `disposition` (`accepted` | `draft`) is the only field that speaks to
  acceptance, and recovery leaves it and `acceptanceDecisionId` untouched;
- a `recovered` success never advances proof, verification or the Outcome's
  `runState`;
- an accepted delivery still required the owner's AcceptanceDecision at request
  time, bound to the same Contract revision and artifact version.

## Destination paths

`destination` and `manifestPath` are the path the owner named. A symlinked
parent directory is followed — by the export when it wrote, and by recovery when
it reads, so the two agree — which means the bytes may sit behind that link
rather than at the literal path shown. This is DLV-02 and is unchanged by this
slice; it is recorded here so a renderer does not present the path as a resolved
location.

Inside the destination, **no symbolic link is followed at all** — not at the
artifact itself and not at any directory on the way to it. Either reports
`DELIVERY_RECOVERY_MISMATCH`. A digest matches identical bytes wherever they
live, so without this a delivery could be confirmed from content that was never
transferred.

## Reconciliation timing

Recovery runs once during daemon start, before the listener serves, and is
idempotent — a second pass finds nothing pending and closes nothing. There is no
endpoint to trigger it and a client must not poll for one; a `pending` row
simply resolves by the time the daemon is answering requests.
