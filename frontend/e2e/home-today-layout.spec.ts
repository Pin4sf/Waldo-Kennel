import { expect, test } from "@playwright/test";

test.use({
	userAgent:
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	viewport: { width: 1512, height: 982 },
});

test("Today keeps the morning-brief completion control inside the Catch Up pane", async ({ page }) => {
	await page.goto("/#/home?homePhase=morning&homeContext=catch_up");

	const pane = page.locator(".home-today-catch-up-pane");
	const finish = page.getByRole("button", { name: "Finish morning brief →" });
	await expect(pane).toBeVisible();
	await expect(finish).toBeVisible();

	const paneBox = await pane.boundingBox();
	const finishBox = await finish.boundingBox();
	expect(paneBox).not.toBeNull();
	expect(finishBox).not.toBeNull();
	expect(finishBox!.y + finishBox!.height).toBeLessThanOrEqual(paneBox!.y + paneBox!.height);
});

test("Today keeps exactly one expanded Quick Capture and no horizontal overflow", async ({ page }) => {
	await page.goto("/#/home?homePhase=afternoon&homeContext=before_next");

	await expect(page.getByRole("textbox", { name: "Quick Capture" })).toHaveCount(1);
	await expect(page.getByRole("heading", { name: "Before your next thing" })).toBeVisible();

	const hasOverflow = await page.evaluate(
		() => document.documentElement.scrollWidth > document.documentElement.clientWidth,
	);
	expect(hasOverflow).toBe(false);
});
