# Superseded: Composed Outcomes program plan

- **Original date:** 2026-08-29
- **Status:** Historical implementation program; retired from active kernel context on 2026-09-04
- **Current authority:** ADR 0007 as consolidated/superseded-in-part, ADR 0008, ADR 0009, and the 2026-09-04 Kennel-builds-Kennel program

The original program implemented the first composed-Outcome vertical and is preserved in Git history for provenance. Do **not** use its old one-WorkUnit / “composition is the execution graph” mechanism as current architecture.

The valid composed-Outcome decisions that survive are recorded directly in [`../../adr/0007-composed-outcomes.md`](../../adr/0007-composed-outcomes.md):

- criterion-bound contribution;
- parent criterion coverage / parent-retained proof;
- child authority may narrow but never widen parent authority;
- stale-parent revision handling;
- independent child proof;
- user-only Acceptance;
- one contributing layer in v1.

ADR 0008 now separates responsibility composition from execution decomposition:

> **Create another Outcome when responsibility splits. Create another WorkUnit when execution splits.**

For current implementation sequencing use:

- [`../../product/kennel-build-program.md`](../../product/kennel-build-program.md)
- [`2026-09-04-kennel-builds-kennel.md`](2026-09-04-kennel-builds-kennel.md)
- [`../../product/kennel-dogfood-acceptance-matrix.md`](../../product/kennel-dogfood-acceptance-matrix.md)

The verbatim 2026-08-29 plan remains available in Git history if an implementation archaeology task specifically needs it.
