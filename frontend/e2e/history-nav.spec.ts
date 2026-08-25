import { expect, test } from "@playwright/test";

// Work routes intentionally use the product shell without titlebar history
// chrome. Home retains the persistent titlebar cluster beside its sidebar.
test("Home retains the titlebar history controls", async ({ page }) => {
	await page.goto("/#/home");
	await expect(page).toHaveURL(/\/home$/);

	const back = page.getByRole("button", { name: "Go back" });
	const forward = page.getByRole("button", { name: "Go forward" });

	await expect(back).toBeVisible();
	await expect(forward).toBeVisible();
	await expect(back).toBeDisabled();
	await expect(forward).toBeDisabled();
});
