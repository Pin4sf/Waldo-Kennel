import { describe, expect, it } from "vitest";
import { homeFixture } from "./home-fixture";
import { resolveHomeDayPhase } from "./home-day-phase";

describe("resolveHomeDayPhase", () => {
  it.each([
    [new Date(2026, 7, 22, 5, 0, 0), "morning"],
    [new Date(2026, 7, 22, 11, 59, 59), "morning"],
    [new Date(2026, 7, 22, 12, 0, 0), "afternoon"],
    [new Date(2026, 7, 22, 16, 59, 59), "afternoon"],
    [new Date(2026, 7, 22, 17, 0, 0), "evening"],
    [new Date(2026, 7, 23, 4, 59, 59), "evening"],
  ] as const)("maps local time %s to %s", (now, expected) => {
    expect(resolveHomeDayPhase(now)).toBe(expected);
  });
});

describe("homeFixture day state", () => {
  it("keeps the default fixture deterministic for existing Home tests", () => {
    const fixture = homeFixture("today");

    expect(fixture.dayPhase).toBe("morning");
    expect(fixture.contextFlow).toBe("catch_up");
    expect(fixture.presentation.briefLabel).toBe("Morning brief");
  });

  it("keeps contextual flow independent from the editorial day phase", () => {
    const fixture = homeFixture("today", "ready", {
      dayPhase: "morning",
      contextFlow: "quiet_focus",
    });

    expect(fixture.dayPhase).toBe("morning");
    expect(fixture.contextFlow).toBe("quiet_focus");
    expect(fixture.presentation.briefLabel).toBe("Morning brief");
  });

  it("uses genuinely different editorial content in each chapter", () => {
    const morning = homeFixture("today", "ready", { dayPhase: "morning" });
    const afternoon = homeFixture("today", "ready", { dayPhase: "afternoon" });
    const evening = homeFixture("today", "ready", { dayPhase: "evening" });

    expect(morning.brief).not.toEqual(afternoon.brief);
    expect(afternoon.brief).not.toEqual(evening.brief);
    expect(morning.presentation.greeting).toBe("Good morning, Shivansh.");
    expect(afternoon.presentation.greeting).toBe("Good afternoon, Shivansh.");
    expect(evening.presentation.greeting).toBe("Good evening, Shivansh.");
  });
});
