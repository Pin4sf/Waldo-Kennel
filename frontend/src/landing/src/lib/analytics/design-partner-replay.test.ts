import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  DESKTOP_PROJECT_KEY,
  REPLAY_PATH,
  replayDecision,
} from "./design-partner-replay";

const OWN_PROJECT = "phc_marketingProjectKeyThatIsNotTheApp";

describe("design partner replay", () => {
  it("records on the design partner page once the site has its own project", () => {
    expect(
      replayDecision({ key: OWN_PROJECT, pathname: REPLAY_PATH, optedOut: false }),
    ).toEqual({ record: true });
  });

  // The whole reason this function exists. Both surfaces point at one project
  // today, and arming replay there would record the screens of desktop installs
  // already in the field, including builds too old to carry the client block.
  it("refuses while the key is the desktop project, on the right page and with consent", () => {
    expect(
      replayDecision({ key: DESKTOP_PROJECT_KEY, pathname: REPLAY_PATH, optedOut: false }),
    ).toEqual({ record: false, reason: "shared-project" });
  });

  it("refuses whitespace-padded variants of the desktop key", () => {
    expect(
      replayDecision({ key: `  ${DESKTOP_PROJECT_KEY}  `, pathname: REPLAY_PATH, optedOut: false }),
    ).toEqual({ record: false, reason: "shared-project" });
  });

  it("records nowhere else on the site", () => {
    for (const path of ["/", "/download", "/docs/overview", "/design-partner", "/design-partners-old"]) {
      expect(
        replayDecision({ key: OWN_PROJECT, pathname: path, optedOut: false }),
      ).toEqual({ record: false, reason: "wrong-path" });
    }
  });

  it("treats a trailing slash and a sub-path as the same page", () => {
    for (const path of [`${REPLAY_PATH}/`, `${REPLAY_PATH}/apply`]) {
      expect(replayDecision({ key: OWN_PROJECT, pathname: path, optedOut: false })).toEqual({
        record: true,
      });
    }
  });

  // Recording somebody who declined analytics is worse than not recording.
  it("refuses when the visitor has not consented", () => {
    expect(
      replayDecision({ key: OWN_PROJECT, pathname: REPLAY_PATH, optedOut: true }),
    ).toEqual({ record: false, reason: "not-consented" });
  });

  it("refuses when no key is configured at all", () => {
    for (const key of [undefined, "", "   "]) {
      expect(replayDecision({ key, pathname: REPLAY_PATH, optedOut: false })).toEqual({
        record: false,
        reason: "no-key",
      });
    }
  });
});

// Kennel intentionally has no inherited analytics project. The retained donor
// site still guards its historical AO key, but it must never equal the desktop
// default or silently reconnect the fork to upstream analytics.
it("keeps the Kennel desktop default empty and isolated from the donor site key", () => {
  const here = dirname(fileURLToPath(import.meta.url));
  const sharedPath = resolve(here, "../../../../shared/posthog-config.ts");
  const source = readFileSync(sharedPath, "utf8");
  const match = source.match(/DEFAULT_POSTHOG_PROJECT_KEY\s*=\s*"([^"]*)"/);
  expect(match, `could not find DEFAULT_POSTHOG_PROJECT_KEY in ${sharedPath}`).toBeTruthy();
  expect(match![1]).toBe("");
  expect(DESKTOP_PROJECT_KEY).not.toBe(match![1]);
});
