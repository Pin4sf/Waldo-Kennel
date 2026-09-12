# Demo integration verification

Date: 2026-09-12

This is a local, unpushed integration candidate based on `origin/beta`
`f2132a83c6a09145cd38234d55097128815de432`. The candidate combines the
completed Issue #115 governed repository-tools work with the existing shell,
onboarding, and Mission UI lane. It does not merge, publish, deploy, or make
an owner AcceptanceDecision.

## Candidate identity

- Branch: `codex/demo-integration-20260912`
- Code candidate HEAD before this verification record: `b2c7ec908a6da2f020806ab71f77fece42cfa275`
- App: `Kennel`, bundle id `in.heywaldo.kennel`, version `0.10.3`
- App bundle: `frontend/out/Kennel-darwin-arm64/Kennel.app`
- Embedded daemon SHA-256: `faa2673d1c2c8a03d47dffc66051a8ad5c49b29590637146e15b071d4a181ede`
- `app.asar` SHA-256: `d8000c5e78f0452f292b058bcb1bb0db2864096761aa4aeebd09162fc6ac9df5`

Launch the disposable candidate with an isolated profile and data directory:

```bash
KENNEL_ELECTRON_DATA_DIR=/tmp/kennel-demo-electron-profile-2 \
KENNEL_DATA_DIR=/tmp/kennel-demo-data-2 \
KENNEL_RUN_FILE=/tmp/kennel-demo-running-2.json \
KENNEL_PORT=43998 \
frontend/out/Kennel-darwin-arm64/Kennel.app/Contents/MacOS/kennel
```

## Desktop journey

Observed through the packaged macOS app and accessibility tree, not through
manual database/API setup:

| Journey | Result |
| --- | --- |
| First-run onboarding and provider selection | Pass |
| Native folder picker, valid Git repository registration, default remote | Pass |
| Settings → Island settings, hide/show toggle, return | Pass |
| Outcome description → editable Contract proposal | Pass |
| Explicit Write workspace and Run local commands authorization | Pass |
| Unverified reasoning state blocks Plan start | Pass; `REASONING_NOT_VERIFIED` was visible |
| Owner-triggered Codex App Server verification | Pass; UI showed “Verified for this provider and model” |
| Planning conversation and clarification | Blocked: remained “Waiting for the agent”; no Plan revision was created |
| Real WorkUnit execution, retained artifact, Evidence/Verification | Not reached; correctly not claimed |
| Ready for review and owner rework/acceptance | Not reached; decision remains open |
| Restart with the same isolated data/profile | Pass: project and saved Contract Outcome reappeared exactly once |
| Mission Control Contract and Prove & Close projection | Pass; evidence fields and explicit owner decision were visible |

The planning daemon exited while the packaged app still showed the waiting
state. Teardown reproduced the known `Object has been destroyed` warning.
This is an open runtime/recovery issue, not evidence of Plan or execution
completion. No acceptance, evidence, verification, delivery, or external
effect was submitted on the owner's behalf.

## Current-code delta context

- The #35 prove/close lane is present in the integrated candidate: Contract
  criteria, evidence and verification entry points, owner decision, rework,
  and delivery remain distinct. The UI leaves acceptance unavailable until
  an explicit owner decision.
- The #78 Mission/attempt lane is present: Board/List Outcomes, selected
  Outcome Mission Control, Contract/Plan/Execution/Evidence tabs, and return
  to the Outcome board are available. A real Attempt/session inspector and
  retained result still require a successfully proposed Plan and governed
  execution.

## Packaging

`npm run bootstrap` and the packaged arm64 Electron build completed. Forge
configuration includes the macOS DMG maker and updater ZIP maker. Signing is
conditional on `APPLE_SIGNING_IDENTITY` or `CSC_LINK`; notarization is
conditional on `KENNEL_NOTARY_PROFILE` or the Apple API key triplet. No
credentials were supplied or read. The candidate is therefore a local
unsigned verification artifact until those release prerequisites are supplied
by an authorized release owner.

## Open acceptance rows

1. Native planning must return a valid Plan proposal and survive provider
   shutdown/restart without ambiguous ownership or duplicate sessions.
2. An approved Plan must run the real scoped WorkUnit and retain its artifact
   and evidence through restart.
3. The owner must review proof, optionally request rework, and decide
   acceptance; Kennel must not infer this decision.
4. Offline/unsupported provider behavior needs a full packaged runtime pass
   beyond the observed unverified-provider gate.
