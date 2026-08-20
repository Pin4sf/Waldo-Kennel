# Domain Glossary

Canonical vocabulary for Waldo Kennel. Terms only — no implementation details,
no decisions (those live in `docs/adr/` and `docs/product/`).

## Product identity and responsibility

- **Waldo** — the user-owned intelligence and responsibility layer. Waldo helps
  the user define what matters, decide what requires attention, preserve
  continuity, and consciously close or release responsibility.

- **Kennel** — Waldo's desktop presence for local work and personal continuity.
  Kennel is not a separate assistant identity.

- **Responsibility Space** — an explicit boundary that groups responsibility,
  context, authority, and retention. Launch kinds are **Work Project** and
  **Personal Home**.

- **Work Project** — a Responsibility Space associated with a local work area in
  which executable Outcomes may be planned and attempted.

- **Personal Home** — the local Responsibility Space for confirmed personal Open
  Loops, Daily Snapshots, and communication items that do not belong to a Work
  Project. It is not durable Memory or a claim to understand the user's whole
  life.

- **Outcome** — a responsibility delegated by the user: something that needs to
  become true. An Outcome is not a task, message thread, provider session,
  commit, pull request, or check result.

- **Contract Revision** — an immutable version of an Outcome's goal, success
  criteria, constraints, review expectation, authority envelope, and stop
  conditions.

- **Plan Revision** — a versioned proposal for turning an Outcome contract into
  bounded work.

- **Mission Map** — the understandable user-facing projection of a Plan Revision.
  It is optional and is not a second responsibility object.

- **Work Unit** — the smallest bounded schedulable part of an Outcome plan.

- **Attempt** — one execution of one Work Unit. A retry or provider handoff is a
  new Attempt, not a rewrite of history.

- **Agent Session** — a provider-native execution conversation used by an
  Attempt. Session completion does not complete or accept an Outcome.

- **Evidence** — provenance-bearing support or contradiction tied to a success
  criterion and the exact subject evaluated.

- **Verification** — an explicit method evaluating Evidence against a current
  criterion. Verification informs but never creates Acceptance.

- **Acceptance** — the user's conscious decision that an Outcome responsibility
  is handled. Accepted history is immutable.

- **Open Loop** — a confirmed unresolved responsibility or commitment that the
  user wants Waldo to preserve and revisit. It may lead to an Outcome, wait on
  another person, or close without executable work.

- **Commitment Candidate** — a proposed responsibility inferred from a message,
  note, observation, or activity. It is not an Open Loop until the user confirms
  or corrects it.

- **Communication Thread** — an external conversation referenced as source
  context. A thread is never the canonical identity of an Outcome or Open Loop.

- **Daily Snapshot** — a time-bounded, correctable summary of trusted facts,
  confirmed Open Loops, and explicit notes. It is a projection, not durable
  Memory or a productivity score.

- **Desktop Observation** — an optional, local, untrusted observation derived
  from user-consented desktop capture.

- **Context Episode** — a correctable grouping of Desktop Observations. It cannot
  become Evidence, an Open Loop, a rule, or Memory without an explicit admission
  decision.

- **Memory Candidate** — a later proposal for durable continuity. It remains
  separate from source observations and requires explicit admission, correction,
  provenance, and deletion controls.

- **Effect Intent** — a frozen proposal to change state outside the current
  bounded local computation, including creating a remote draft, pull request, or
  message.

- **Effect Receipt** — the reconciled record of what an Effect Intent actually
  changed, including an explicit unknown result when certainty is unavailable.

- **Needs You** — a derived attention state for irreducible user judgment between
  materially different valid paths.

- **Action Required** — a derived attention state for one exact human-only action.

- **Waiting** — a derived attention state for an unresolved dependency or timed
  condition when immediate user action is not useful.

- **Ready for Acceptance** — a derived Outcome state indicating that required
  Evidence and Verification exist for the current contract revision. It is not
  Acceptance.

- **Re-entry** — restoration of the minimum exact context needed to continue an
  Open Loop or create a successor Outcome without mutating accepted history.

## Daemon surfaces

- **Loopback Listener** — the daemon's original, always-on HTTP surface bound to
  `127.0.0.1`. Serves the desktop app and CLI. Unauthenticated: safe only because
  nothing off-box can reach it. Behaviour is unchanged by the mobile work.

- **LAN Listener** — a second, network-facing HTTP surface the daemon exposes so a
  physical phone can reach it. Off by default; exists only while **Connect Mobile**
  is enabled. Authenticated (see **Connection Password**). Serves the app API but
  **not** daemon-control routes (shutdown/lifecycle stay Loopback-only).

## Mobile connectivity

- **Connect Mobile** — the user-facing capability that opens the **LAN Listener**
  and lets a paired phone use the app over the local network. A desktop toggle;
  when off, no LAN surface exists.

- **Connection Password** — the single, rotating secret that authorises a phone on
  the **LAN Listener**. 8-char alphanumeric, generated by the desktop, shown to the
  user out-of-band (read off the screen), typed into the phone. Rotating it drops
  the currently-connected phone. It is the _sole_ secret — it is never carried in
  the **Pairing QR**.

- **Pairing QR** — a scannable code that carries only the **LAN Listener** address
  (`host` + `port`) and a schema version. Non-secret by design: it conveys _where_
  to connect, never the **Connection Password**. A photo of it alone cannot connect.

- **Pairing** — the act of a phone acquiring a working connection: scan the
  **Pairing QR** (or type host/port manually), then enter the **Connection
  Password** in a popup. A successful pairing is remembered on the phone.

- **Paired Phone** — the phone currently authorised against the active **Connection
  Password**. Only one is effectively connected at a time (single rotating password).

- **Lockout** — the brute-force guard on the **LAN Listener**: repeated failed
  password attempts from a source are throttled (per-source, not global, so a
  hostile device cannot lock out the real phone). Resets on a successful auth.

- **Home-network-only** — the trust boundary the feature assumes. Transport is
  unencrypted, so the **LAN Listener** is only safe on a network the user trusts;
  the desktop UI states this plainly.
