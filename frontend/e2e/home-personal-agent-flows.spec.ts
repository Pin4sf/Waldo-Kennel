import { expect, test } from "@playwright/test";

test.use({
	userAgent:
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	viewport: { width: 1512, height: 982 },
});

test("morning Catch Up preserves exact provenance return and one Quick Capture", async ({ page }) => {
	await page.goto("/#/home?homePhase=morning&homeContext=catch_up");

	await expect(page.getByRole("heading", { name: "Good morning, Shivansh." })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Catch Up" })).toBeVisible();
	await expect(page.getByRole("textbox", { name: "Quick Capture" })).toHaveCount(1);

	const inspect = page.getByRole("button", { name: "Inspect source" });
	await inspect.click();
	await expect(page.getByRole("dialog", { name: "Source provenance" })).toBeVisible();
	await page.keyboard.press("Escape");
	await expect(inspect).toBeFocused();
});

test("afternoon Today recalibrates before the next commitment", async ({ page }) => {
	await page.goto("/#/home?homePhase=afternoon&homeContext=before_next");

	await expect(page.getByRole("heading", { name: "Good afternoon, Shivansh." })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Before your next thing" })).toBeVisible();
	await expect(page.getByText("Pricing workshop", { exact: true })).toBeVisible();
	await expect(page.getByPlaceholder("Note what changed…")).toBeVisible();
});

test("evening Today reaches a truthful Daily Close receipt and History", async ({ page }) => {
	await page.goto("/#/home?homePhase=evening&homeContext=evening_review");

	await expect(page.getByRole("heading", { name: "Good evening, Shivansh." })).toBeVisible();
	await page.getByRole("link", { name: "Start Closure" }).click();
	await expect(page.getByRole("heading", { name: "Close the day deliberately" })).toBeVisible();
	await page.getByRole("button", { name: "Review complete — preview Daily Close" }).click();
	await expect(page.getByRole("heading", { name: "Daily Close preview" })).toBeVisible();
	await expect(page.getByText("Preview receipt — nothing was saved or closed")).toBeVisible();
	await page.getByRole("link", { name: "Inspect History" }).click();
	await expect(page.getByRole("heading", { name: "History" })).toBeVisible();
});

test("Open Loops explains responsibility and keeps Work handoff preview-only", async ({ page }) => {
	await page.goto("/#/home/open-loops");

	await page.getByRole("button", { name: "Deck follow-up" }).click();
	await expect(page.getByText("Prepare the revised deck for Ashish; do not send it yet.")).toBeVisible();
	await page.getByRole("button", { name: "Continue in Work" }).click();
	await expect(page.getByRole("status", { name: "Open Loop preview status" })).toContainText(
		"No Work Outcome or responsibility link has been created",
	);
	await expect(page.getByRole("textbox", { name: "Quick Capture" })).toHaveCount(0);
});

test("Memory Review rejects only the local candidate preview", async ({ page }) => {
	await page.goto("/#/home/memory");

	await expect(page.getByText("Candidate — not memory")).toBeVisible();
	await page.getByRole("button", { name: "Reject" }).click();
	await expect(page.getByRole("status", { name: "Memory review status" })).toContainText(
		"Rejected in this preview — no durable memory changed",
	);
	await expect(page.getByRole("textbox", { name: "Quick Capture" })).toHaveCount(0);
});

test("Home and Work return to the last meaningful route in each mode", async ({ page }) => {
	await page.goto("/#/home/open-loops");

	await page.getByRole("button", { name: "Work", exact: true }).click();
	await expect(page).toHaveURL(/\/#\/$/);
	await page.getByRole("button", { name: "Home", exact: true }).click();
	await expect(page).toHaveURL(/\/#\/home\/open-loops$/);
});

test("narrow Home replaces the index with detail and returns exact focus", async ({ page }) => {
	await page.setViewportSize({ width: 720, height: 760 });
	await page.goto("/#/home/open-loops");

	const row = page.getByRole("button", { name: "Deck follow-up" });
	await row.click();
	const back = page.getByRole("button", { name: "Back to Open Loops" });
	await expect(back).toBeVisible();
	await back.click();
	await expect(row).toBeFocused();

	const hasOverflow = await page.evaluate(
		() => document.documentElement.scrollWidth > document.documentElement.clientWidth,
	);
	expect(hasOverflow).toBe(false);
});

test("Home never exposes Work project or session navigation", async ({ page }) => {
	await page.setViewportSize({ width: 900, height: 760 });
	await page.goto("/#/home/history");

	await expect(page.getByRole("navigation", { name: "Home destinations" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Today" })).toBeVisible();
	await expect(page.getByRole("link", { name: "Open Loops" })).toBeVisible();
	await expect(page.getByRole("link", { name: /projects/i })).toHaveCount(0);
	await expect(page.getByRole("link", { name: /sessions/i })).toHaveCount(0);

	const hasOverflow = await page.evaluate(
		() => document.documentElement.scrollWidth > document.documentElement.clientWidth,
	);
	expect(hasOverflow).toBe(false);
});
