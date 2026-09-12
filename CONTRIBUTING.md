# Contributing

We welcome focused contributions: code, documentation, triage, examples and
tests. GitHub issues and pull requests are the durable coordination record.
For non-trivial work, open or comment on an issue before implementation so the
scope, authority documents and shared-file ownership are clear.

## Ways to contribute

| Type             | Examples                                       |
| ---------------- | ---------------------------------------------- |
| Code             | Fixes, features, adapters, performance         |
| Docs             | README, `docs/`, architecture notes            |
| Triage           | Repro bugs, tighten reports, label suggestions |
| Examples / tests | Recipes, edge cases, flaky-test hunts          |

## Quick start

1. **Read the contract** — [AGENTS.md](AGENTS.md) (layout, commands, hard rules, PR hygiene)
2. **Check current truth** — [docs/STATUS.md](docs/STATUS.md), then use [ROADMAP.md](ROADMAP.md) for direction
3. **Pick something focused** — [open issues](https://github.com/Pin4sf/Waldo-Kennel/issues); prefer an assigned issue in the current milestone
4. **Claim it** — comment `I'd like to work on this` and wait for assignment
5. **Open a clear PR** — narrow change, link the issue, user-visible impact, tests
6. **Iterate** — address review; maintainers merge

Need the product/run overview first? Start with [README.md](README.md),
[docs/architecture.md](docs/architecture.md), and
[docs/development.md](docs/development.md).

Two onboarding notes matter on current `main`:

- On fresh Linux setups, prefer `cd frontend && npm run package` unless you have also installed distro packaging tools such as `rpm`/`rpmbuild` for `npm run make`.
- Mobile companion app docs are still being filled in. Do not assume `packages/mobile/README.md` is a complete headless setup guide on this branch.

### Bugs and features

Use the GitHub issue forms (**Bug report** / **Feature request**) so reports stay reproducible.
Bug reports should include the Kennel version or commit, environment, repro steps, and expected vs actual behavior.

Feature proposals should describe the user Outcome and current limitation before
suggesting implementation. Roadmap milestones are not assignments; keep each
issue to one falsifiable slice and cite the governing ADR/spec when applicable.

### Pull requests

Follow **PR hygiene** in [AGENTS.md](AGENTS.md): one issue per PR, conventional commits, explicit dependencies and shared-file ownership, intentional omissions, and verification evidence.

Kennel uses `beta` as its integration branch:

1. Refresh `origin/beta` before starting an assigned issue.
2. Create one issue branch from that commit; do not commit directly to `beta` or `main`.
3. Open the implementation PR against `beta` and wait for its required tests and review.
4. Coordinate shared files, generated API artifacts, and migration numbers with the integration owner before editing.
5. Maintainers test the integrated `beta` branch and promote it through a separate `beta` -> `main` PR.

Releases are cut only from tested `main` by the designated release conductor. A merge to `beta` is not a release or deployment authorization.

## Code of Conduct

Be respectful, constructive and assume good intent. Keep technical disagreement
focused on evidence, behavior and product invariants. Do not put credentials,
private repository content or vulnerability details in a public issue.

Thanks for making Waldo Kennel better for the next person who shows up.
