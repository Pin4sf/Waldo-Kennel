import { expect, test } from "@playwright/test";

test.use({
  userAgent:
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
  viewport: { width: 1512, height: 982 },
});

test("Home gives Waldo the context region and restores Catch Up exactly", async ({ page }) => {
  await page.goto("/#/home?homePhase=morning&homeContext=catch_up");

  const launcher = page.getByRole("button", { name: "Open Waldo" });
  await expect(page.getByRole("heading", { name: "Catch Up" })).toBeVisible();
  await launcher.click();

  await expect(page.getByRole("region", { name: "Waldo" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Catch Up" })).toHaveCount(0);
  await expect(page.getByRole("textbox", { name: "Quick Capture" })).toHaveCount(1);

  await page.getByRole("button", { name: "Close Waldo" }).click();
  await expect(page.getByRole("heading", { name: "Catch Up" })).toBeVisible();
  await expect(launcher).toBeFocused();
});

test("the configured shortcut opens the same global Waldo rail", async ({ page }) => {
  await page.goto("/#/home");
  await expect(page.getByRole("button", { name: "Open Waldo" })).toBeVisible();

  await page.keyboard.press("Meta+Shift+Space");

  await expect(page.getByRole("region", { name: "Waldo" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Open Waldo" })).toHaveAttribute(
    "aria-expanded",
    "true",
  );
});

test("Conversation episodes preserve context and semantic boundaries", async ({ page }) => {
  await page.goto("/#/home");
  await page.getByRole("button", { name: "Open Waldo" }).click();

  await expect(page.getByRole("status", { name: "Waldo preview boundary" })).toContainText(
    "No model or agent is running",
  );
  await expect(page.getByRole("tab", { name: "Conversation" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await expect(page.getByText("Contextual", { exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Observation preview" })).toContainText(
    "Candidate",
  );
  await expect(page.getByRole("region", { name: "Proposal preview" })).toContainText(
    "Approval required",
  );
  await page.getByRole("button", { name: "Review proposed command" }).click();
  await expect(page.getByRole("status", { name: "Proposal preview status" })).toContainText(
    "Nothing was created, sent, or saved",
  );

  await page.getByRole("button", { name: "Detach context" }).click();
  await expect(page.getByText("No attached context")).toBeVisible();

  await page.getByRole("button", { name: "Launch readiness" }).click();
  const result = page.getByRole("region", { name: "Result preview" });
  await expect(result).toContainText("Result ready");
  await expect(result).toContainText("Outcome · Unknown");
  await page.getByRole("button", { name: "Show result detail" }).click();
  await expect(result).toContainText("No verification or AcceptanceDecision exists");
});

test("Activity exposes one bounded delegated run under Waldo", async ({ page }) => {
  await page.goto("/#/home");
  await page.getByRole("button", { name: "Open Waldo" }).click();

  const conversationTab = page.getByRole("tab", { name: "Conversation" });
  await conversationTab.focus();
  await page.keyboard.press("ArrowRight");
  await expect(page.getByRole("tab", { name: "Activity" })).toBeFocused();

  const activity = page.getByRole("region", { name: "Agent activity preview" });
  await expect(activity).toContainText("Running");
  await expect(activity).toContainText("Current step");
  await page.getByRole("button", { name: "Inspect run" }).click();
  await expect(activity).toContainText("Bounded research specialist");
  await expect(activity).toContainText("No external messages, mutations, or acceptance");
  await expect(activity).toContainText("Calendar, decision note, current Work context");
  await expect(activity).toContainText("Returns to Work · no Outcome created");
  await expect(activity).toContainText("Result ready · Outcome · Unknown");
  await expect(activity).not.toContainText("Verified outcome");
});

test("Home exposes completed continuity screens without promoting them to primary", async ({ page }) => {
  await page.goto("/#/home");

  const primary = page.getByRole("group", { name: "Primary Home destinations" });
  await expect(primary.getByRole("link", { name: "Today" })).toBeVisible();
  await expect(primary.getByRole("link", { name: "Open Loops" })).toBeVisible();

  const continuity = page.getByRole("group", { name: "Review and continuity" });
  await continuity.getByRole("link", { name: "Daily Close" }).click();
  await expect(page.getByRole("heading", { name: "Close the day deliberately" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Quick Capture" })).toHaveCount(0);

  await page.getByRole("link", { name: "Memory Review" }).click();
  await expect(page.getByRole("heading", { name: "Memory Review" })).toBeVisible();
  await page.getByRole("link", { name: "History" }).click();
  await expect(page.getByRole("heading", { name: "History" })).toBeVisible();
});

test("narrow Waldo becomes a full-content layer with a visible Back action", async ({ page }) => {
  // Electron enforces a 960px minimum window width. The center panel is still
  // narrow after the Home sidebar, so this is the real desktop boundary.
  await page.setViewportSize({ width: 960, height: 760 });
  await page.goto("/#/home/open-loops");
  await page.getByRole("button", { name: "Open Waldo" }).click();

  const rail = page.getByRole("region", { name: "Waldo" });
  await expect(rail).toBeVisible();
  await expect(rail.getByText("Back", { exact: true })).toBeVisible();

  const railBox = await rail.boundingBox();
  const centerBox = await page.locator(".center-panel-surface").boundingBox();
  expect(railBox).not.toBeNull();
  expect(centerBox).not.toBeNull();
  expect(Math.abs(railBox!.width - centerBox!.width)).toBeLessThanOrEqual(2);

  const hasOverflow = await page.evaluate(
    () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
  );
  expect(hasOverflow).toBe(false);
});

test("Work opens one Waldo right region and retains the Home Work switch", async ({ page }) => {
  await page.goto("/#/work");

  await page.getByRole("button", { name: "Open Waldo" }).click();

  await expect(page.getByRole("region", { name: "Waldo" })).toHaveCount(1);
  await expect(page.getByRole("navigation", { name: "Waldo mode" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Work", exact: true })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
});
