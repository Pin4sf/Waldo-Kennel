import { expect, test } from "@playwright/test";

test.use({
	userAgent:
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	viewport: { width: 1512, height: 982 },
});

test("Today keeps the morning-brief completion control inside the Catch Up pane", async ({ page }) => {
	await page.goto("/#/home");

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
